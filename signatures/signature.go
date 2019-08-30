package signatures

import (
	"fmt"
	"time"

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

//TODO: these should be ENUMs in protobufs
const (
	KeyTypeBLSGroupSig = "BLS"
	KeyTypeSecp256k1   = "secp256k1"
)

var nullAddr = common.BytesToAddress([]byte{})

// Ownership is also included in signature and
// is used to create an address suitable for
// ChainTree ownership (that can also include conditions)
type Ownership struct {
	// Type is an ENUM of the supported signature types
	Type string
	//PublicKey is optional when on a signature, some signatures contain a public key
	PublicKey []byte
	// Conditions is an optional parens script
	Conditions string
}

type Signature struct {
	*Ownership
	// Signers is an optional array of counts of the public keys of signers
	// used for aggregated BLS signatures where the public keys are known out-of-band of signatures
	Signers []uint32
	// The actual signature bytes
	Signature []byte
	// PreImage is an optional field used for the hash-preimage condition
	PreImage string
}

func (o *Ownership) Address() (common.Address, error) {
	// in the case of conditions, all signatures are treated similarly to produce an address
	// and we just take the hash of the public key and the conditions and produce an address
	if o.Conditions != "" {
		pubKeyWithConditions := append(o.PublicKey, []byte(o.Conditions)...)
		return bytesToAddress(pubKeyWithConditions), nil
	}

	switch o.Type {
	case KeyTypeSecp256k1:
		key, err := crypto.UnmarshalPubkey(o.PublicKey)
		if err != nil {
			xerrors.Errorf("error unmarshaling public key: %w", err)
		}
		return crypto.PubkeyToAddress(*key), nil
	case KeyTypeBLSGroupSig:
		return bytesToAddress(o.PublicKey), nil
	default:
		return nullAddr, xerrors.Errorf("unknown keytype: %s", o.Type)
	}
}

func bytesToAddress(bits []byte) common.Address {
	return common.BytesToAddress(crypto.Keccak256(bits)[12:])
}

func (s *Signature) RestorePublicKey(hsh []byte) error {
	if s.Type != KeyTypeSecp256k1 {
		return xerrors.Errorf("error only KeyTypeSecp256k1 supports key recovery")
	}
	recoveredPub, err := crypto.SigToPub(hsh, s.Signature)
	if err != nil {
		return fmt.Errorf("error recovering signature: %v", err)
	}
	s.PublicKey = crypto.FromECDSAPub(recoveredPub)
	return nil
}

func (s *Signature) Valid(hsh []byte, scope parens.Scope) (bool, error) {
	if len(s.PublicKey) == 0 {
		return false, xerrors.Errorf("public key was 0, perhaps you forgot to restore it from sig?")
	}
	if scope == nil {
		scope = parens.NewScope(defaultScope)
	}
	conditionsValid, err := s.validConditions(scope)
	if err != nil {
		return false, xerrors.Errorf("error validating conditions: %w", err)
	}
	if !conditionsValid {
		return false, nil
	}

	switch s.Type {
	case KeyTypeSecp256k1:
		return crypto.VerifySignature(s.PublicKey, hsh, s.Signature[:len(s.Signature)-1]), nil
	case KeyTypeBLSGroupSig:
		verKey := bls.BytesToVerKey(s.PublicKey)
		verified, err := verKey.Verify(s.Signature, hsh)
		if err != nil {
			logger.Warningf("error verifying signature: %v", err)
			return false, xerrors.Errorf("error verifying: %w", err)
		}
		return verified, nil
	default:
		return false, xerrors.Errorf("Unknown key type %s", s.Type)
	}
}

func (s *Signature) validConditions(scope parens.Scope) (bool, error) {
	if s.Conditions == "" {
		return true, nil
	}
	scope.Bind("hashed-preimage", crypto.Keccak256Hash([]byte(s.PreImage)).String())

	res, err := parens.ExecuteStr(s.Conditions, scope)
	if err != nil {
		return false, xerrors.Errorf("error executing script: %w", err)
	}
	if res == true {
		return true, nil
	}

	logger.Debugf("conditions for signature failed")
	return false, nil
}
