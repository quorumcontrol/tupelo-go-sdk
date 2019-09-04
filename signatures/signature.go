package signatures

import (
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

func RestorePublicKey(s *signatures.Signature, hsh []byte) error {
	if s.Ownership.Type != signatures.Ownership_KeyTypeSecp256k1 {
		return xerrors.Errorf("error only KeyTypeSecp256k1 supports key recovery")
	}
	recoveredPub, err := crypto.SigToPub(hsh, s.Signature)
	if err != nil {
		return fmt.Errorf("error recovering signature: %v", err)
	}
	s.Ownership.PublicKey = crypto.FromECDSAPub(recoveredPub)
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
