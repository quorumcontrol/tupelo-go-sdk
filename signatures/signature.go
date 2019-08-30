package signatures

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/xerrors"

	"github.com/spy16/parens"
	"github.com/spy16/parens/stdlib"
)

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
	stdlib.RegisterMath(scope)
	defaultScope = scope
}

//TODO: these should be ENUMs in protobufs
const (
	KeyTypeBLSGroupSig = "BLS"
	KeyTypeSecp256k1   = "secp256k1"
)

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
	// Hash is an optional field used for the hash-preimage condition
	Hash []byte
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
		return common.BytesToAddress([]byte{}), xerrors.Errorf("unknown keytype: %s", o.Type)
	}
}

func bytesToAddress(bits []byte) common.Address {
	return common.BytesToAddress(crypto.Keccak256(bits)[12:])
}
