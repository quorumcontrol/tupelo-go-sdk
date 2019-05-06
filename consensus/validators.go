package consensus

import (
	"fmt"

	cid "github.com/ipfs/go-cid"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/messages/transactions"

	extmsgs "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
)

func getReceiveTokenPayloads(txns []*transactions.Transaction) ([]*transactions.ReceiveTokenPayload, error) {
	receiveTokens := make([]*transactions.ReceiveTokenPayload, 0)
	for _, t := range txns {
		if t.Type == transactions.Transaction_RECEIVETOKEN {
			rt := &transactions.ReceiveTokenPayload{}
			err := typecaster.ToType(t.Payload, rt)
			if err != nil {
				return nil, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error typecasting payload: %v", err)}
			}
			receiveTokens = append(receiveTokens, rt)
		}
	}
	return receiveTokens, nil
}

// Validator functions

// IsTokenRecipient is only applicable to RECEIVE_TOKEN transactions
// it checks whether the destination chaintree id matches our id or not
func IsTokenRecipient(tree *dag.Dag, blockWithHeaders *chaintree.BlockWithHeaders) (bool, chaintree.CodedError) {
	// first determine if there are any RECEIVE_TOKEN transactions in here
	receiveTokens, err := getReceiveTokenPayloads(blockWithHeaders.Transactions)
	if err != nil {
		return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting RECEIVE_TOKEN transactions: %v", err)}
	}

	if len(receiveTokens) == 0 {
		// if no RECEIVE_TOKEN transactions are present, short-circuit to valid
		return true, nil
	}

	// we have at least one RECEIVE_TOKEN; make sure it was intended for this chaintree

	id, _, err := tree.Resolve([]string{"id"})
	if err != nil {
		return false, &ErrorCode{Memo: fmt.Sprintf("error: %v", err), Code: ErrUnknown}
	}

	for _, rt := range receiveTokens {
		senderDag, codedErr := getSenderDagFromReceive(rt)
		if codedErr != nil {
			return false, codedErr
		}

		tokenName, codedErr := getTokenNameFromReceive(senderDag)
		if codedErr != nil {
			return false, codedErr
		}

		sendToken, codedErr := getSendTokenFromReceive(senderDag, tokenName)
		if codedErr != nil {
			return false, codedErr
		}

		if id.(string) != sendToken.Destination {
			return false, nil
		}
	}

	return true, nil
}

// isValidSignature checks payloads with both Signature and Tip elements to
// verify that the Signature is indeed valid for Tip. It is currently used for
// RECEIVE_TOKEN transactions and so looks for them explicitly, but should be
// generalized to other transaction types that have similar Signature & Tip
// elements when/if they appear.
//
// GenerateIsValidSignature is a higher-order function that takes a signature
// verifier function arg and returns an IsValidSignature validator function
// (see above) that calls the given sigVerifier with the Signature and Tip it
// receives and uses its return values to determine validity.
func GenerateIsValidSignature(sigVerifier func(sig *extmsgs.Signature) (bool, error)) chaintree.BlockValidatorFunc  {
	isValidSignature := func(tree *dag.Dag, blockWithHeaders *chaintree.BlockWithHeaders) (bool, chaintree.CodedError) {
		// first determine if there any RECEIVE_TOKEN transactions in here
		receiveTokens, err := getReceiveTokenPayloads(blockWithHeaders.Transactions)
		if err != nil {
			return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting RECEIVE_TOKEN transactions: %v", err)}
		}

		if len(receiveTokens) == 0 {
			// if no RECEIVE_TOKEN transactions are present, short-circuit to valid
			return true, nil
		}

		// we have at least one RECEIVE_TOKEN transaction; make sure Signature is valid for Tip
		for _, rt := range receiveTokens {
			sig := &rt.Signature

			tip, err := cid.Cast(rt.Tip)
			if err != nil {
				return false, &ErrorCode{Code: ErrInvalidTip, Memo: fmt.Sprintf("error casting tip to CID: %v", err)}
			}

			sigNewTip, err := cid.Cast(sig.NewTip)
			if err != nil {
				return false, &ErrorCode{Code: ErrInvalidTip, Memo: fmt.Sprintf("error casting tip to CID: %v", err)}
			}

			if sigNewTip != tip {
				return false, nil
			}

			valid, err := sigVerifier(sig)
			if err != nil {
				return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error verifying signature: %v", err)}
			}

			if !valid {
				return false, nil
			}
		}

		return true, nil
	}

	return isValidSignature
}
