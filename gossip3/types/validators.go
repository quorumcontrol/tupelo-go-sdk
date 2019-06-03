package types

import (
	"context"
	"fmt"
	"strings"

	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/chaintree/typecaster"

	cid "github.com/ipfs/go-cid"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/messages/build/go/transactions"

	"github.com/quorumcontrol/tupelo-go-sdk/conversion"
	extmsgs "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
)

func getReceiveTokenPayloads(txns []*transactions.Transaction) ([]*transactions.ReceiveTokenPayload, error) {
	receiveTokens := make([]*transactions.ReceiveTokenPayload, 0)
	for _, t := range txns {
		if t.Type == transactions.Transaction_RECEIVETOKEN {
			rt, err := t.EnsureReceiveTokenPayload()
			if err != nil {
				return nil, &consensus.ErrorCode{Code: consensus.ErrUnknown, Memo: fmt.Sprintf("error reading payload: %v", err)}
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
		return false, &consensus.ErrorCode{Code: consensus.ErrUnknown, Memo: fmt.Sprintf("error getting RECEIVE_TOKEN transactions: %v", err)}
	}

	if len(receiveTokens) == 0 {
		// if no RECEIVE_TOKEN transactions are present, short-circuit to valid
		return true, nil
	}

	// we have at least one RECEIVE_TOKEN; make sure it was intended for this chaintree

	id, _, err := tree.Resolve([]string{"id"})
	if err != nil {
		return false, &consensus.ErrorCode{Memo: fmt.Sprintf("error: %v", err), Code: consensus.ErrUnknown}
	}

	for _, rt := range receiveTokens {
		senderDag, codedErr := consensus.GetSenderDagFromReceive(rt)
		if codedErr != nil {
			return false, codedErr
		}

		tokenName, codedErr := consensus.GetTokenNameFromReceive(senderDag)
		if codedErr != nil {
			return false, codedErr
		}

		sendToken, codedErr := consensus.GetSendTokenFromReceive(senderDag, tokenName)
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
func GenerateIsValidSignature(sigVerifier func(sig *extmsgs.Signature) (bool, error)) chaintree.BlockValidatorFunc {
	isValidSignature := func(tree *dag.Dag, blockWithHeaders *chaintree.BlockWithHeaders) (bool, chaintree.CodedError) {
		// first determine if there any RECEIVE_TOKEN transactions in here
		receiveTokens, err := getReceiveTokenPayloads(blockWithHeaders.Transactions)
		if err != nil {
			return false, &consensus.ErrorCode{Code: consensus.ErrUnknown, Memo: fmt.Sprintf("error getting RECEIVE_TOKEN transactions: %v", err)}
		}

		if len(receiveTokens) == 0 {
			// if no RECEIVE_TOKEN transactions are present, short-circuit to valid
			return true, nil
		}

		// we have at least one RECEIVE_TOKEN transaction; make sure Signature is valid for Tip
		for _, rt := range receiveTokens {
			sig, err := conversion.ToExternalSignature(rt.Signature)
			if err != nil {
				return false, &consensus.ErrorCode{Code: consensus.ErrInvalidSig, Memo: fmt.Sprintf("error converting signature: %v", err)}
			}

			tip, err := cid.Cast(rt.Tip)
			if err != nil {
				return false, &consensus.ErrorCode{Code: consensus.ErrInvalidTip, Memo: fmt.Sprintf("error casting tip to CID: %v", err)}
			}

			sigNewTip, err := cid.Cast(sig.NewTip)
			if err != nil {
				return false, &consensus.ErrorCode{Code: consensus.ErrInvalidTip, Memo: fmt.Sprintf("error casting tip to CID: %v", err)}
			}

			if sigNewTip != tip {
				return false, nil
			}

			valid, err := sigVerifier(sig)
			if err != nil {
				return false, &consensus.ErrorCode{Code: consensus.ErrUnknown, Memo: fmt.Sprintf("error verifying signature: %v", err)}
			}

			if !valid {
				return false, nil
			}
		}

		return true, nil
	}

	return isValidSignature
}

func isTokenBurn(tokenName string, burnAmount uint64, tx *transactions.Transaction) bool {
	amount := uint64(0)
	if burnAmount == 0 {
		amount = 1
	} else {
		amount = burnAmount
	}
	return tx.Type == transactions.Transaction_SENDTOKEN &&
		tx.SendTokenPayload.Name == tokenName &&
		tx.SendTokenPayload.Amount >= amount &&
		tx.SendTokenPayload.Destination == ""
}

// HasBurnGenerator is a higher-order generator (because we need notary group config for the token name) that returns a
// ChainTree validator that looks for the precense of a token burn in a block of transactions
// if the block has a transaction which has a SendToken with the config TransactionToken name,
// a value > 0 and a "" destination, it is considered a burn.
func HasBurnGenerator(ctx context.Context, ng *NotaryGroup) (chaintree.BlockValidatorFunc, error) {
	tokenName := ng.Config().TransactionToken
	if tokenName == "" {
		return nil, fmt.Errorf("error you must specify a TransactionCurrency to use the HasBurnGenerator")
	}

	burnAmount := ng.Config().BurnAmount

	var burnValidator chaintree.BlockValidatorFunc = func(tree *dag.Dag, blockWithHeaders *chaintree.BlockWithHeaders) (bool, chaintree.CodedError) {
		for _, tx := range blockWithHeaders.Block.Transactions {
			if isTokenBurn(tokenName, burnAmount, tx) {
				return true, nil
			}
		}
		return false, nil
	}
	return burnValidator, nil
}

// IsOwner is a chaintree BlockValidator that looks at the block headers and makes sure
// an authorized owner (in the consensus.TreePathForAuthentications path) has signed
// this block.
func IsOwner(tree *dag.Dag, blockWithHeaders *chaintree.BlockWithHeaders) (bool, chaintree.CodedError) {
	id, _, err := tree.Resolve([]string{"id"})
	if err != nil {
		return false, &consensus.ErrorCode{Memo: fmt.Sprintf("error: %v", err), Code: consensus.ErrUnknown}
	}

	headers := &consensus.StandardHeaders{}

	err = typecaster.ToType(blockWithHeaders.Headers, headers)
	if err != nil {
		return false, &consensus.ErrorCode{Memo: fmt.Sprintf("error: %v", err), Code: consensus.ErrUnknown}
	}

	var addrs []string

	uncastAuths, _, err := tree.Resolve(strings.Split("tree/"+consensus.TreePathForAuthentications, "/"))
	if err != nil {
		return false, &consensus.ErrorCode{Code: consensus.ErrUnknown, Memo: fmt.Sprintf("err resolving: %v", err)}
	}
	// If there are no authentications then the Chain Tree is still owned by its genesis key
	if uncastAuths == nil {
		addrs = []string{consensus.DidToAddr(id.(string))}
	} else {
		err = typecaster.ToType(uncastAuths, &addrs)
		if err != nil {
			return false, &consensus.ErrorCode{Code: consensus.ErrUnknown, Memo: fmt.Sprintf("err casting: %v", err)}
		}
	}

	for _, addr := range addrs {
		isSigned, err := consensus.IsBlockSignedBy(blockWithHeaders, addr)
		if err != nil {
			return false, &consensus.ErrorCode{Memo: fmt.Sprintf("error finding if signed: %v", err), Code: consensus.ErrUnknown}
		}

		if isSigned {
			return true, nil
		}
	}

	return false, nil
}
