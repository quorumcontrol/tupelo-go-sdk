package signatures

import (
	"crypto/ecdsa"
	"fmt"
	"math"
	"time"

	"github.com/quorumcontrol/messages/build/go/signatures"

	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/xerrors"

	"github.com/spy16/parens"
	"github.com/spy16/parens/stdlib"
)

var logger = logging.Logger("signatures")

var defaultScope parens.Scope

type entry struct {
	key string
	val interface{}
}

func bindMany(scope parens.Scope, entries []entry) error {
	var err error
	for _, entry := range entries {
		err = scope.Bind(entry.key, entry.val)
		if err != nil {
			return xerrors.Errorf("error binding: %w", err)
		}
	}
	return nil
}

func init() {
	scope := parens.NewScope(nil)
	err := bindMany(scope, []entry{
		{"cond", parens.MacroFunc(stdlib.Conditional)},
		{"true", true},
		{"false", false},
		{"nil", false},
		{"println", func(str string) {
			fmt.Println(str)
		}},
		{"now", func() int64 {
			return time.Now().UTC().Unix()
		}},
	})
	if err != nil {
		panic(err)
	}
	err = stdlib.RegisterMath(scope)
	if err != nil {
		panic(err)
	}
	defaultScope = scope
}

var nullAddr = common.BytesToAddress([]byte{})

func Address(o *signatures.Ownership) (common.Address, error) {
	// in the case of conditions, all signatures are treated similarly to produce an address
	// and we just take the hash of the public key and the conditions and produce an address
	if o.Conditions != "" {
		pubKeyWithConditions := append(o.PublicKey, []byte(o.Conditions)...)
		return bytesToAddress(pubKeyWithConditions), nil
	}

	switch o.Type {
	case signatures.Ownership_KeyTypeSecp256k1:
		key, err := crypto.UnmarshalPubkey(o.PublicKey)
		if err != nil {
			return nullAddr, xerrors.Errorf("error unmarshaling public key: %w", err)
		}
		return crypto.PubkeyToAddress(*key), nil
	case signatures.Ownership_KeyTypeBLSGroupSig:
		return bytesToAddress(o.PublicKey), nil
	default:
		return nullAddr, xerrors.Errorf("unknown keytype: %s", o.Type)
	}
}

func bytesToAddress(bits []byte) common.Address {
	return common.BytesToAddress(crypto.Keccak256(bits)[12:])
}

func RestoreEcdsaPublicKey(s *signatures.Signature, hsh []byte) error {
	if s.Ownership.Type != signatures.Ownership_KeyTypeSecp256k1 {
		return xerrors.Errorf("error only KeyTypeSecp256k1 supports key recovery")
	}
	recoveredPub, err := crypto.SigToPub(hsh, s.Signature)
	if err != nil {
		return xerrors.Errorf("error recovering signature: %w", err)
	}
	s.Ownership.PublicKey = crypto.FromECDSAPub(recoveredPub)
	return nil
}

func RestoreBLSPublicKey(s *signatures.Signature, knownVerKeys []*bls.VerKey) error {
	if len(knownVerKeys) != len(s.Signers) {
		return xerrors.Errorf("error known verkeys length did not match signers length: %d != %d", len(knownVerKeys), len(s.Signers))
	}
	var verKeys []*bls.VerKey

	var signerCount uint64
	for i, cnt := range s.Signers {
		if cnt > 0 {
			signerCount++
			verKey := knownVerKeys[i]
			newKeys := make([]*bls.VerKey, cnt)
			for j := uint32(0); j < cnt; j++ {
				newKeys[j] = verKey
			}
			verKeys = append(verKeys, newKeys...)
		}
	}
	key, err := bls.SumVerKeys(verKeys)
	if err != nil {
		return xerrors.Errorf("error summing keys: %w", err)
	}
	s.Ownership.PublicKey = key.Bytes()
	return nil
}

func Valid(s *signatures.Signature, hsh []byte, scope parens.Scope) (bool, error) {
	if len(s.Ownership.PublicKey) == 0 {
		return false, xerrors.Errorf("public key was 0, perhaps you forgot to restore it from sig?")
	}
	if scope == nil {
		scope = parens.NewScope(defaultScope)
	}
	conditionsValid, err := validConditions(s, scope)
	if err != nil {
		return false, xerrors.Errorf("error validating conditions: %w", err)
	}
	if !conditionsValid {
		return false, nil
	}

	switch s.Ownership.Type {
	case signatures.Ownership_KeyTypeSecp256k1:
		return crypto.VerifySignature(s.Ownership.PublicKey, hsh, s.Signature[:len(s.Signature)-1]), nil
	case signatures.Ownership_KeyTypeBLSGroupSig:
		verKey := bls.BytesToVerKey(s.Ownership.PublicKey)
		verified, err := verKey.Verify(s.Signature, hsh)
		if err != nil {
			logger.Warningf("error verifying signature: %v", err)
			return false, xerrors.Errorf("error verifying: %w", err)
		}
		return verified, nil
	default:
		return false, xerrors.Errorf("Unknown key type %s", s.Ownership.Type)
	}
}

func validConditions(s *signatures.Signature, scope parens.Scope) (bool, error) {
	if s.Ownership.Conditions == "" {
		return true, nil
	}
	err := scope.Bind("hashed-preimage", func() string {
		return crypto.Keccak256Hash([]byte(s.PreImage)).String()
	})
	if err != nil {
		return false, xerrors.Errorf("error binding: %w", err)
	}

	res, err := parens.ExecuteStr(s.Ownership.Conditions, scope)
	if err != nil {
		return false, xerrors.Errorf("error executing script: %w", err)
	}
	if res == true {
		return true, nil
	}

	logger.Debugf("conditions for signature failed")
	return false, nil
}

func EcdsaToOwnership(key *ecdsa.PublicKey) *signatures.Ownership {
	return &signatures.Ownership{
		Type:      signatures.Ownership_KeyTypeSecp256k1,
		PublicKey: crypto.FromECDSAPub(key),
	}
}

func BLSToOwnership(key *bls.VerKey) *signatures.Ownership {
	return &signatures.Ownership{
		Type:      signatures.Ownership_KeyTypeBLSGroupSig,
		PublicKey: key.Bytes(),
	}
}

func SignerCount(sig *signatures.Signature) int {
	signerCount := 0
	for _, sigCount := range sig.Signers {
		if sigCount > 0 {
			signerCount++
		}
	}
	return signerCount
}

func BLSSign(key *bls.SignKey, hsh []byte, signerLen, signerIndex int) (*signatures.Signature, error) {
	if signerIndex >= signerLen {
		return nil, xerrors.Errorf("signer index must be less than signer length i: %d, l: %d", signerLen, signerIndex)
	}
	verKey, err := key.VerKey()
	if err != nil {
		return nil, xerrors.Errorf("error getting verkey: %w", err)
	}
	sig, err := key.Sign(hsh)
	if err != nil {
		return nil, xerrors.Errorf("Error signing: %w", err)
	}

	signers := make([]uint32, signerLen)
	signers[signerIndex] = 1
	return &signatures.Signature{
		Ownership: BLSToOwnership(verKey),
		Signers:   signers,
		Signature: sig,
	}, nil
}

func AggregateBLSSignatures(sigs []*signatures.Signature) (*signatures.Signature, error) {
	signerCount := len(sigs[0].Signers)
	newSig := &signatures.Signature{
		Ownership: &signatures.Ownership{
			Type: signatures.Ownership_KeyTypeBLSGroupSig,
		},
		Signers: make([]uint32, signerCount),
	}
	sigsToAggregate := make([][]byte, len(sigs))
	pubKeysToAggregate := make([]*bls.VerKey, len(sigs))
	for i, sig := range sigs {
		if sig.Ownership.Type != signatures.Ownership_KeyTypeBLSGroupSig {
			return nil, xerrors.Errorf("wrong signature type, can only aggregate BLS signatures")
		}
		if len(sig.Signers) != signerCount {
			return nil, xerrors.Errorf("all signatures to aggregate must have the same signer length %d != %d", len(sig.Signers), signerCount)
		}
		sigsToAggregate[i] = sig.Signature
		pubKeysToAggregate[i] = bls.BytesToVerKey(sig.Ownership.PublicKey)
		for i, cnt := range sig.Signers {
			if existing := newSig.Signers[i]; cnt > math.MaxUint32-existing || existing > math.MaxUint32-cnt {
				return nil, xerrors.Errorf("error would overflow: %d %d", cnt, existing)
			}
			newSig.Signers[i] += cnt
		}
	}

	aggregateSig, err := bls.SumSignatures(sigsToAggregate)
	if err != nil {
		return nil, xerrors.Errorf("error summing signatures: %w", err)
	}
	newSig.Signature = aggregateSig

	aggregatePublic, err := bls.SumVerKeys(pubKeysToAggregate)
	if err != nil {
		return nil, xerrors.Errorf("error aggregating verKeys: %w", err)
	}
	newSig.Ownership.PublicKey = aggregatePublic.Bytes()
	return newSig, nil
}
