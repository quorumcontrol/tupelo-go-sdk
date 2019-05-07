package testfakes

import (
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/messages/signatures"
	"github.com/quorumcontrol/messages/transactions"
)

func SetOwnershipTransaction(keyAddr string) *transactions.Transaction {
	payload := &transactions.SetOwnershipPayload{
		Authentication: []string{keyAddr},
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_SETOWNERSHIP,
		Payload: &transactions.Transaction_SetOwnershipPayload{SetOwnershipPayload: payload},
	}
}

func SetDataTransaction(path, value string) *transactions.Transaction {
	payload := &transactions.SetDataPayload{
		Path:  path,
		Value: []byte(value),
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_SETDATA,
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
		Type:    transactions.Transaction_ESTABLISHTOKEN,
		Payload: &transactions.Transaction_EstablishTokenPayload{EstablishTokenPayload: payload},
	}
}

func MintTokenTransaction(name string, amount uint64) *transactions.Transaction {
	payload := &transactions.MintTokenPayload{
		Name:   name,
		Amount: amount,
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_MINTTOKEN,
		Payload: &transactions.Transaction_MintTokenPayload{MintTokenPayload: payload},
	}
}

func SendTokenTransaction(id, name string, amount uint64, destination string) *transactions.Transaction {
	payload := &transactions.SendTokenPayload{
		Id:          id,
		Name:        name,
		Amount:      amount,
		Destination: destination,
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_SENDTOKEN,
		Payload: &transactions.Transaction_SendTokenPayload{SendTokenPayload: payload},
	}
}

func ReceiveTokenTransaction(sendTid string, tip []byte, sig *signatures.Signature, leaves [][]byte) *transactions.Transaction {
	payload := &transactions.ReceiveTokenPayload{
		SendTokenTransactionId: sendTid,
		Tip:       tip,
		Signature: sig,
		Leaves:    leaves,
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_RECEIVETOKEN,
		Payload: &transactions.Transaction_ReceiveTokenPayload{ReceiveTokenPayload: payload},
	}
}

func NewValidUnsignedTransactionBlock(txn *transactions.Transaction) chaintree.BlockWithHeaders {
	return chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}
}
