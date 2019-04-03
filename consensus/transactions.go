package consensus

import (
	"fmt"
	"strings"

	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/typecaster"
)

const (
	TreePathForAuthentications = "_tupelo/authentications"
	TreePathForTokens          = "_tupelo/tokens"
	TreePathForStake           = "_tupelo/stake"
	TreePathForData            = "data"

	TokenMintLabel             = "mints"
	TokenSendLabel             = "sends"
	TokenReceiveLabel          = "receives"
)

func init() {
	typecaster.AddType(SetDataPayload{})
	typecaster.AddType(SetOwnershipPayload{})
	typecaster.AddType(EstablishTokenPayload{})
	typecaster.AddType(MintTokenPayload{})
	typecaster.AddType(SendTokenPayload{})
	typecaster.AddType(Token{})
	typecaster.AddType(TokenMonetaryPolicy{})
	typecaster.AddType(TokenMint{})
	typecaster.AddType(TokenSend{})
	typecaster.AddType(StakePayload{})
	cbornode.RegisterCborType(SetDataPayload{})
	cbornode.RegisterCborType(SetOwnershipPayload{})
	cbornode.RegisterCborType(EstablishTokenPayload{})
	cbornode.RegisterCborType(MintTokenPayload{})
	cbornode.RegisterCborType(SendTokenPayload{})
	cbornode.RegisterCborType(Token{})
	cbornode.RegisterCborType(TokenMonetaryPolicy{})
	cbornode.RegisterCborType(TokenMint{})
	cbornode.RegisterCborType(TokenSend{})
	cbornode.RegisterCborType(StakePayload{})
}

// SetDataPayload is the payload for a SetDataTransaction
// Path / Value
type SetDataPayload struct {
	Path  string
	Value interface{}
}

func complexType(obj interface{}) bool {
	switch obj.(type) {
	// These are the built in type of go (excluding map) plus cid.Cid
	// Use SetAsLink if attempting to set map
	case bool, byte, complex64, complex128, error, float32, float64, int, int8, int16, int32, int64, string, uint, uint16, uint32, uint64, uintptr, cid.Cid, *bool, *byte, *complex64, *complex128, *error, *float32, *float64, *int, *int8, *int16, *int32, *int64, *string, *uint, *uint16, *uint32, *uint64, *uintptr, *cid.Cid, []bool, []byte, []complex64, []complex128, []error, []float32, []float64, []int, []int8, []int16, []int32, []int64, []string, []uint, []uint16, []uint32, []uint64, []uintptr, []cid.Cid, []*bool, []*byte, []*complex64, []*complex128, []*error, []*float32, []*float64, []*int, []*int8, []*int16, []*int32, []*int64, []*string, []*uint, []*uint16, []*uint32, []*uint64, []*uintptr, []*cid.Cid:
		return false
	default:
		return true
	}
}

func DecodePath(path string) ([]string, error) {
	trimmed := strings.TrimPrefix(path, "/")

	if trimmed == "" {
		return []string{}, nil
	}

	split := strings.Split(trimmed, "/")
	for _, component := range split {
		if component == "" {
			return nil, fmt.Errorf("malformed path string containing repeated separator: %s", path)
		}
	}

	return split, nil
}

// SetDataTransaction just sets a path in tree/data to arbitrary data.
func SetDataTransaction(tree *dag.Dag, transaction *chaintree.Transaction) (newTree *dag.Dag, valid bool, codedErr chaintree.CodedError) {
	payload := &SetDataPayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error casting payload: %v", err)}
	}

	path, err := DecodePath(payload.Path)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error decoding path: %v", err)}
	}

	// SET_DATA always sets inside tree/data
	dataPath, _ := DecodePath(TreePathForData)
	path = append(dataPath, path...)

	if complexType(payload.Value) {
		newTree, err = tree.SetAsLink(path, payload.Value)
	} else {
		newTree, err = tree.Set(path, payload.Value)
	}
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error setting: %v", err)}
	}

	return newTree, true, nil
}

type SetOwnershipPayload struct {
	Authentication []string
}

// SetOwnershipTransaction changes the ownership of a tree by adding a public key array to /_tupelo/authentications
func SetOwnershipTransaction(tree *dag.Dag, transaction *chaintree.Transaction) (newTree *dag.Dag, valid bool, codedErr chaintree.CodedError) {
	payload := &SetOwnershipPayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error casting payload: %v", err)}
	}

	path, err := DecodePath(TreePathForAuthentications)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error decoding path: %v", err)}
	}

	newTree, err = tree.SetAsLink(path, payload.Authentication)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error setting: %v", err)}
	}

	return newTree, true, nil
}

type Token struct {
	MonetaryPolicy *cid.Cid
	Mints          *cid.Cid
	Sends          *cid.Cid
	Receives       *cid.Cid
}

type TokenMonetaryPolicy struct {
	Maximum uint64
}

type EstablishTokenPayload struct {
	Name           string
	MonetaryPolicy TokenMonetaryPolicy
}

func EstablishTokenTransaction(tree *dag.Dag, transaction *chaintree.Transaction) (newTree *dag.Dag, valid bool, codedErr chaintree.CodedError) {
	payload := &EstablishTokenPayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error typecasting payload: %v", err)}
	}

	tokenName := payload.Name

	ledger := NewTreeLedger(tree, tokenName)

	tokenExists, err := ledger.TokenExists()
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error checking for existence of token \"%s\"", tokenName)}
	}
	if tokenExists {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error, token \"%s\" already exists", tokenName)}
	}

	newTree, err = ledger.CreateToken(payload.MonetaryPolicy)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: err.Error()}
	}

	return newTree, true, nil
}

type MintTokenPayload struct {
	Name   string
	Amount uint64
}

func MintTokenTransaction(tree *dag.Dag, transaction *chaintree.Transaction) (newTree *dag.Dag, valid bool, codedErr chaintree.CodedError) {
	payload := &MintTokenPayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error typecasting payload: %v", err)}
	}

	tokenName := payload.Name
	ledger := NewTreeLedger(tree, tokenName)

	newTree, err = ledger.MintToken(payload.Amount)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error minting token: %v", err)}
	}

	return newTree, true, nil
}

type SendTokenPayload struct {
	Id          string
	Name        string
	Amount      uint64
	Destination string
}

func SendTokenTransaction(tree *dag.Dag, transaction *chaintree.Transaction) (newTree *dag.Dag, valid bool, codedErr chaintree.CodedError) {
	payload := &SendTokenPayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error typecasting payload: %v", err)}
	}

	tokenName := payload.Name

	ledger := NewTreeLedger(tree, tokenName)

	newTree, err = ledger.SendToken(payload.Id, payload.Destination, payload.Amount)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error sending token: %v", err)}
	}

	return newTree, true, nil
}

type ReceiveTokenPayload struct {
	SendTokenTransactionId string
	Tip cid.Cid
	Signature Signature
	Leaves map[string]cid.Cid
}

func ReceiveTokenTransaction(tree *dag.Dag, transaction *chaintree.Transaction) (newTree *dag.Dag, valid bool, codedError chaintree.CodedError) {
	payload := &ReceiveTokenPayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error setting: %v", err)}
	}
	// FIXME: placeholder return to satisfy typechecker for now
	return tree, true, nil
}

type StakePayload struct {
	GroupId string
	Amount  uint64
	DstKey  PublicKey
	VerKey  PublicKey
}

// THIS IS A pre-ALPHA TRANSACTION AND NO RULES ARE ENFORCED! Anyone can stake and join a group with no consequences.
// additionally, it only allows staking a single group at the moment
func StakeTransaction(tree *dag.Dag, transaction *chaintree.Transaction) (newTree *dag.Dag, valid bool, codedErr chaintree.CodedError) {
	payload := &StakePayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error casting payload: %v", err)}
	}

	path, err := DecodePath(TreePathForStake)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error decoding path: %v", err)}
	}

	newTree, err = tree.SetAsLink(path, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error setting: %v", err)}
	}

	return newTree, true, nil
}
