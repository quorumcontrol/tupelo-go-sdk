package consensus

import (
	"fmt"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/typecaster"
)

// Validator functions


// isTokenRecipient is only applicable to RECEIVE_TOKEN transactions
// it checks whether the destination chaintree id matches our id or not
func IsTokenRecipient(tree *dag.Dag, blockWithHeaders *chaintree.BlockWithHeaders) (bool, chaintree.CodedError) {
	// first determine if there are any RECEIVE_TOKEN transactions in here
	receiveTokens := make([]*ReceiveTokenPayload, 0)
	for _, t := range blockWithHeaders.Transactions {
		if t.Type == TransactionTypeReceiveToken {
			rt := &ReceiveTokenPayload{}
			err := typecaster.ToType(t.Payload, rt)
			if err != nil {
				return false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error typecasting payload: %v", err)}
			}
			receiveTokens = append(receiveTokens, rt)
		}
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

	headers := &StandardHeaders{}

	err = typecaster.ToType(blockWithHeaders.Headers, headers)
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
