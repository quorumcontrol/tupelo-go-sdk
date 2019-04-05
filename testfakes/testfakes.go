package testfakes

import (
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/messages/transactions"
)

func SetOwnershipTransaction(keyAddr string) *transactions.Transaction {
	payload := &transactions.SetOwnershipPayload{
		Authentication: []string{keyAddr},
	}

	return &transactions.Transaction{
		Type:    transactions.TransactionType_SetOwnership,
		Payload: &transactions.Transaction_SetOwnershipPayload{SetOwnershipPayload: payload},
	}
}

func SetDataTransaction(path, value string) *transactions.Transaction {
	payload := &transactions.SetDataPayload{
		Path:  path,
		Value: []byte(value),
	}

	return &transactions.Transaction{
		Type:    transactions.TransactionType_SetData,
		Payload: &transactions.Transaction_SetDataPayload{SetDataPayload: payload},
	}
}

func EstablishTokenTransaction(name string, max uint64) *transactions.Transaction {
	policy := &transactions.TokenMonetaryPolicy{Maximum: max}

	payload := &transactions.EstablishTokenPayload{
		Name:           name,
		MonetaryPolicy: policy,
	}

	return &transactions.Transaction{
		Type:    transactions.TransactionType_EstablishToken,
		Payload: &transactions.Transaction_EstablishTokenPayload{EstablishTokenPayload: payload},
	}
}

func MintTokenTransaction(name string, amount uint64) *transactions.Transaction {
	payload := &transactions.MintTokenPayload{
		Name:   name,
		Amount: amount,
	}

	return &transactions.Transaction{
		Type:    transactions.TransactionType_MintToken,
		Payload: &transactions.Transaction_MintTokenPayload{MintTokenPayload: payload},
	}
}

func NewValidUnsignedTransactionBlock(txn *transactions.Transaction) *chaintree.BlockWithHeaders {
	return &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}
}
