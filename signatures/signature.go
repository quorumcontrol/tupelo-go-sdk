package signatures

import (
	"crypto/ecdsa"
	"fmt"
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

func init() {
	scope := parens.NewScope(nil)
	scope.Bind("cond", parens.MacroFunc(stdlib.Conditional))
	scope.Bind("true", true)
	scope.Bind("false", false)
	scope.Bind("nil", false) // nil is falsy
	// TODO: get rid of this
	scope.Bind("println", func(str string) {
		fmt.Println(str)
	})
	scope.Bind("now", func() int64 {
		return time.Now().UTC().Unix()
	})
	stdlib.RegisterMath(scope)
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
			xerrors.Errorf("error unmarshaling public key: %w", err)
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
	scope.Bind("hashed-preimage", func() string {
		return crypto.Keccak256Hash([]byte(s.PreImage)).String()
	})

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
