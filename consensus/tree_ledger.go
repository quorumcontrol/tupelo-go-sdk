package consensus

import (
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/typecaster"
)

var transactionTypes = map[string][]string{
	"credit": {TokenMintLabel, TokenReceiveLabel},
	"debit":  {TokenSendLabel},
}

type TokenLedger interface {
	TokenExists() (bool, error)
	Balance() (uint64, error)
	CreateToken(monetaryPolicy TokenMonetaryPolicy) (*dag.Dag, error)
	MintToken(amount uint64) (*dag.Dag, error)
	SendToken(txId, destination string, amount uint64) (*dag.Dag, error)
}

type TreeLedger struct {
	tokenName string
	tree *dag.Dag
}

var _ TokenLedger = TreeLedger{}

func NewTreeLedger(tree *dag.Dag, tokenName string) *TreeLedger {
	return &TreeLedger{
		tokenName: tokenName,
		tree: tree,
	}
}

func (l TreeLedger) tokenPath() ([]string, error) {
	rootTokenPath, err := DecodePath(TreePathForTokens)
	if err != nil {
		return nil, fmt.Errorf("error, unable to decode tree path for tokens: %v", err)
	}
	return append(rootTokenPath, l.tokenName), nil
}

func (l TreeLedger) tokenTransactionCidsForType(txType string) ([]cid.Cid, error) {
	tokenPath, err := l.tokenPath()
	if err != nil {
		return nil, err
	}

	uncastCids, _, err := l.tree.Resolve(append(tokenPath, txType))
	if err != nil {
		return nil, fmt.Errorf("error fetching %s at %v: %v", txType, tokenPath, err)
	}

	var cids []cid.Cid

	if uncastCids == nil {
		cids = make([]cid.Cid, 0)
	} else {
		cids = make([]cid.Cid, len(uncastCids.([]interface{})))
		for k, c := range uncastCids.([]interface{}) {
			cids[k] = c.(cid.Cid)
		}
	}

	return cids, nil
}

func (l TreeLedger) tokenTransactionCids() (map[string][]cid.Cid, error) {
	allCids := make(map[string][]cid.Cid)

	for _, txTypes := range transactionTypes {
		for _, txType := range txTypes {
			cids, err := l.tokenTransactionCidsForType(txType)
			if err != nil {
				return nil, err
			}
			allCids[txType] = cids
		}
	}

	return allCids, nil
}

func (l TreeLedger) sumTokenTransactions(cids []cid.Cid) (uint64, error) {
	var sum uint64

	for _, c := range cids {
		node, err := l.tree.Get(c)

		if err != nil {
			return 0, fmt.Errorf("error fetching node %v: %v", c, err)
		}

		amount, _, err := node.Resolve([]string{"amount"})

		if err != nil {
			return 0, fmt.Errorf("error fetching amount from %v: %v", node, err)
		}

		sum = sum + amount.(uint64)
	}

	return sum, nil
}

func (l TreeLedger) calculateTokenBalance(transactionCids map[string][]cid.Cid) (uint64, error) {
	var balance uint64

	for _, t := range transactionTypes["credit"] {
		sum, err := l.sumTokenTransactions(transactionCids[t])
		if err != nil {
			return 0, err
		}
		balance += sum
	}

	for _, t := range transactionTypes["debit"] {
		sum, err := l.sumTokenTransactions(transactionCids[t])
		if err != nil {
			return 0, err
		}
		balance -= sum
	}

	return balance, nil
}

func (l TreeLedger) Balance() (uint64, error) {
	cids, err := l.tokenTransactionCids()
	if err != nil {
		return 0, err
	}

	return l.calculateTokenBalance(cids)
}

func (l TreeLedger) TokenExists() (bool, error) {
	tokenPath, err := l.tokenPath()
	if err != nil {
		return false, fmt.Errorf("error getting token path: %v", err)
	}

	existingToken, _, err := l.tree.Resolve(tokenPath)
	if err != nil {
		return false, fmt.Errorf("error attempting to resolve %v: %v", tokenPath, err)
	}

	return existingToken != nil, nil
}

func (l TreeLedger) CreateToken(monetaryPolicy TokenMonetaryPolicy) (*dag.Dag, error) {
	tokenPath, err := l.tokenPath()
	if err != nil {
		return nil, fmt.Errorf("error getting token path: %v", err)
	}

	newTree, err := l.tree.SetAsLink(tokenPath, &Token{})
	if err != nil {
		return nil, fmt.Errorf("error creating new token: %v", err)
	}

	newTree, err = newTree.SetAsLink(append(tokenPath, "monetaryPolicy"), monetaryPolicy)
	if err != nil {
		return nil, fmt.Errorf("error setting monetary policy: %v", err)
	}

	return newTree, nil
}

func (l TreeLedger) MintToken(amount uint64) (*dag.Dag, error) {
	if amount == 0 {
		return nil, fmt.Errorf("error, must mint amount greater than 0")
	}

	tokenPath, err := l.tokenPath()
	if err != nil {
		return nil, fmt.Errorf("error getting token path: %v", err)
	}

	monetaryPolicyPath := append(tokenPath, "monetaryPolicy")
	uncastMonetaryPolicy, _, err := l.tree.Resolve(monetaryPolicyPath)
	if err != nil {
		return nil, fmt.Errorf("error fetching monetary policy at path %v: %v", monetaryPolicyPath, err)
	}
	if uncastMonetaryPolicy == nil {
		return nil, fmt.Errorf("error, token at path %v is missing a monetary policy", tokenPath)
	}

	monetaryPolicy := &TokenMonetaryPolicy{}
	err = typecaster.ToType(uncastMonetaryPolicy, monetaryPolicy)
	if err != nil {
		return nil, fmt.Errorf("error typecasting monetary policy: %v", err)
	}

	mintCids, err := l.tokenTransactionCidsForType(TokenMintLabel)
	if err != nil {
		return nil, err
	}

	if monetaryPolicy.Maximum > 0 {
		currentMintedTotal, err := l.sumTokenTransactions(mintCids)
		if err != nil {
			return nil, fmt.Errorf("error summing token mints: %v", err)
		}
		if (currentMintedTotal + amount) > monetaryPolicy.Maximum {
			return nil, fmt.Errorf("new mint would violate monetaryPolicy of maximum: %v", monetaryPolicy.Maximum)
		}
	}

	newMint, err := l.tree.CreateNode(&TokenMint{
		Amount: amount,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create new node: %v", err)
	}

	mintCids = append(mintCids, newMint.Cid())

	newTree, err := l.tree.SetAsLink(append(tokenPath, TokenMintLabel), mintCids)
	if err != nil {
		return nil, fmt.Errorf("error setting: %v", err)
	}

	return newTree, nil
}

func (l TreeLedger) SendToken(txId, destination string, amount uint64) (*dag.Dag, error) {
	// TODO: verify destination is chaintree address?

	if amount == 0 {
		return nil, fmt.Errorf("error, must send an amount greater than 0")
	}

	tokenPath, err := l.tokenPath()
	if err != nil {
		return nil, err
	}

	availableBalance, err := l.Balance()
	if err != nil {
		return nil, err
	}

	if availableBalance < amount {
		return nil, fmt.Errorf("cannot send token, balance of %d is too low to send %d", availableBalance, amount)
	}

	newSend, err := l.tree.CreateNode(&TokenSend{
		Id:          txId,
		Amount:      amount,
		Destination: destination,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create new node: %v", err)
	}

	tokenTxCids, err := l.tokenTransactionCids()

	sentCids := tokenTxCids[TokenSendLabel]
	sentCids = append(sentCids, newSend.Cid())

	newTree, err := l.tree.SetAsLink(append(tokenPath, TokenSendLabel), sentCids)
	if err != nil {
		return nil, fmt.Errorf("error setting: %v", err)
	}

	return newTree, nil
}
