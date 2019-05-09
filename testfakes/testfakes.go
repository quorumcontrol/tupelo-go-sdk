package testfakes

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/quorumcontrol/messages/signatures"
	"github.com/quorumcontrol/messages/transactions"
)

func SetOwnershipTransaction(keyAddr string) *transactions.Transaction {
	payload := &transactions.SetOwnershipPayload{
		Authentication: []string{keyAddr},
	}

	payloadWrapper, err := ptypes.MarshalAny(payload)
	if err != nil {
		panic(err)
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_SETOWNERSHIP,
		Payload: payloadWrapper,
	}
}

func SetDataTransaction(path, value string) *transactions.Transaction {
	payload := &transactions.SetDataPayload{
		Path:  path,
		Value: []byte(value),
	}

	payloadWrapper, err := ptypes.MarshalAny(payload)
	if err != nil {
		panic(err)
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_SETDATA,
		Payload: payloadWrapper,
	}
}

func EstablishTokenTransaction(name string, max uint64) *transactions.Transaction {
	policy := &transactions.TokenMonetaryPolicy{Maximum: max}

	payload := &transactions.EstablishTokenPayload{
		Name:           name,
		MonetaryPolicy: policy,
	}

	payloadWrapper, err := ptypes.MarshalAny(payload)
	if err != nil {
		panic(err)
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_ESTABLISHTOKEN,
		Payload: payloadWrapper,
	}
}

func MintTokenTransaction(name string, amount uint64) *transactions.Transaction {
	payload := &transactions.MintTokenPayload{
		Name:   name,
		Amount: amount,
	}

	payloadWrapper, err := ptypes.MarshalAny(payload)
	if err != nil {
		panic(err)
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_MINTTOKEN,
		Payload: payloadWrapper,
	}
}

func SendTokenTransaction(id, name string, amount uint64, destination string) *transactions.Transaction {
	payload := &transactions.SendTokenPayload{
		Id:          id,
		Name:        name,
		Amount:      amount,
		Destination: destination,
	}

	payloadWrapper, err := ptypes.MarshalAny(payload)
	if err != nil {
		panic(err)
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_SENDTOKEN,
		Payload: payloadWrapper,
	}
}

func ReceiveTokenTransaction(sendTid string, tip []byte, sig *signatures.Signature, leaves [][]byte) *transactions.Transaction {
	payload := &transactions.ReceiveTokenPayload{
		SendTokenTransactionId: sendTid,
		Tip:       tip,
		Signature: sig,
		Leaves:    leaves,
	}

	payloadWrapper, err := ptypes.MarshalAny(payload)
	if err != nil {
		panic(err)
	}

	return &transactions.Transaction{
		Type:    transactions.Transaction_RECEIVETOKEN,
		Payload: payloadWrapper,
	}
}
