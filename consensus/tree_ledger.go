package consensus

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/messages/build/go/transactions"
)

const (
	MonetaryPolicyLabel = "monetaryPolicy"
	TokenBalanceLabel   = "balance"
)

var transactionTypes = map[string][]string{
	"credit": {TokenMintLabel, TokenReceiveLabel},
	"debit":  {TokenSendLabel},
}

type TokenLedger interface {
	TokenExists() (bool, error)
	Balance() (uint64, error)
	EstablishToken(monetaryPolicy transactions.TokenMonetaryPolicy) (*dag.Dag, error)
	MintToken(amount uint64) (*dag.Dag, error)
	SendToken(txId, destination string, amount uint64) (*dag.Dag, error)
	ReceiveToken(sendTokenTxId string, amount uint64) (*dag.Dag, error)
}

type TreeLedger struct {
	tokenName *TokenName
	tree      *dag.Dag
}

var _ TokenLedger = &TreeLedger{}

type Token struct {
	MonetaryPolicy *cid.Cid
	Mints          *cid.Cid
	Sends          *cid.Cid
	Receives       *cid.Cid
	Balance        uint64
}

func NewTreeLedger(tree *dag.Dag, tokenName *TokenName) *TreeLedger {
	return &TreeLedger{
		tokenName: tokenName,
		tree:      tree,
	}
}

func TokenPath(tokenName *TokenName) ([]string, error) {
	l := NewTreeLedger(nil, tokenName)
	return l.tokenPath()
}

func (l *TreeLedger) tokenPath() ([]string, error) {
	rootTokenPath, err := DecodePath(TreePathForTokens)
	if err != nil {
		return nil, fmt.Errorf("error, unable to decode tree path for tokens: %v", err)
	}
	return append(rootTokenPath, l.tokenName.String()), nil
}

func TokenTransactionCidsForType(tree *dag.Dag, tokenName *TokenName, txType string) ([]cid.Cid, error) {
	treePathForTokens, err := DecodePath(TreePathForTokens)
	if err != nil {
		return nil, fmt.Errorf("error, unable to decode tree path for tokens: %v", err)
	}

	path := append(treePathForTokens, tokenName.String(), txType)
	return transactionCidsForPath(tree, path)
}

func transactionCidsForPath(tree *dag.Dag, path []string) ([]cid.Cid, error) {
	uncastCids, _, err := tree.Resolve(context.TODO(), path)
	if err != nil {
		return nil, fmt.Errorf("error resolving path %v: %v", path, err)
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

func (l *TreeLedger) tokenTransactionCidsForType(txType string) ([]cid.Cid, error) {
	tokenPath, err := l.tokenPath()
	if err != nil {
		return nil, err
	}

	tokenPath = append(tokenPath, txType)

	cids, err := transactionCidsForPath(l.tree, tokenPath)
	if err != nil {
		return nil, fmt.Errorf("error fetching %s at %v: %v", txType, tokenPath, err)
	}

	return cids, nil
}

func (l *TreeLedger) tokenTransactionCids() (map[string][]cid.Cid, error) {
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

func (l *TreeLedger) sumTokenTransactions(cids []cid.Cid) (uint64, error) {
	var sum uint64

	for _, c := range cids {
		node, err := l.tree.Get(context.TODO(), c)

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

func (l *TreeLedger) Balance() (uint64, error) {
	tokenPath, err := l.tokenPath()
	if err != nil {
		return 0, err
	}

	balancePath := append(tokenPath, TokenBalanceLabel)
	balanceObj, remaining, err := l.tree.Resolve(context.TODO(), balancePath)
	if err != nil {
		return 0, err
	}

	if len(remaining) > 0 {
		return 0, fmt.Errorf("error resolving token balance: path elements remaining: %v", remaining)
	}

	balance, ok := balanceObj.(uint64)
	if !ok {
		return 0, fmt.Errorf("error resolving token balance; node type (%T) is not a uint64", balanceObj)
	}

	return balance, nil
}

func (l *TreeLedger) TokenExists() (bool, error) {
	tokenPath, err := l.tokenPath()
	if err != nil {
		return false, fmt.Errorf("error getting token path: %v", err)
	}

	existingToken, _, err := l.tree.Resolve(context.TODO(), tokenPath)
	if err != nil {
		return false, fmt.Errorf("error attempting to resolve %v: %v", tokenPath, err)
	}

	return existingToken != nil, nil
}

func (l *TreeLedger) createToken() (*dag.Dag, error) {
	tokenPath, err := l.tokenPath()
	if err != nil {
		return nil, fmt.Errorf("error getting token path: %v", err)
	}

	newTree, err := l.tree.SetAsLink(context.TODO(), tokenPath, &Token{})
	if err != nil {
		return nil, fmt.Errorf("error creating new token: %v", err)
	}

	newTree, err = newTree.Set(context.TODO(), append(tokenPath, TokenBalanceLabel), uint64(0))
	if err != nil {
		return nil, fmt.Errorf("error setting balance: %v", err)
	}

	l.tree = newTree

	return newTree, nil
}

func (l *TreeLedger) EstablishToken(monetaryPolicy transactions.TokenMonetaryPolicy) (*dag.Dag, error) {
	newTree, err := l.createToken()
	if err != nil {
		return nil, err
	}

	tokenPath, err := l.tokenPath()
	if err != nil {
		return nil, err
	}

	newTree, err = newTree.SetAsLink(context.TODO(), append(tokenPath, MonetaryPolicyLabel), monetaryPolicy)
	if err != nil {
		return nil, fmt.Errorf("error setting monetary policy: %v", err)
	}

	l.tree = newTree

	return newTree, nil
}

type TokenMint struct {
	Amount uint64
}

func (l *TreeLedger) MintToken(amount uint64) (*dag.Dag, error) {
	if amount == 0 {
		return nil, fmt.Errorf("error, must mint amount greater than 0")
	}

	tokenPath, err := l.tokenPath()
	if err != nil {
		return nil, fmt.Errorf("error getting token path: %v", err)
	}

	monetaryPolicyPath := append(tokenPath, MonetaryPolicyLabel)
	uncastMonetaryPolicy, _, err := l.tree.Resolve(context.TODO(), monetaryPolicyPath)
	if err != nil {
		return nil, fmt.Errorf("error fetching monetary policy at path %v: %v", monetaryPolicyPath, err)
	}
	if uncastMonetaryPolicy == nil {
		return nil, fmt.Errorf("error, token at path %v is missing a monetary policy", tokenPath)
	}

	monetaryPolicy := &transactions.TokenMonetaryPolicy{}
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

	newMint, err := l.tree.CreateNode(context.TODO(), &TokenMint{
		Amount: amount,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create new node: %v", err)
	}

	mintCids = append(mintCids, newMint.Cid())

	newTree, err := l.tree.SetAsLink(context.TODO(), append(tokenPath, TokenMintLabel), mintCids)
	if err != nil {
		return nil, fmt.Errorf("error setting: %v", err)
	}

	currBalance, err := l.Balance()
	if err != nil {
		return nil, fmt.Errorf("error getting current balance: %v", err)
	}

	newBalance := currBalance + amount
	newTree, err = newTree.Set(context.TODO(), append(tokenPath, TokenBalanceLabel), newBalance)
	if err != nil {
		return nil, fmt.Errorf("error updating balance: %v", err)
	}

	l.tree = newTree

	return newTree, nil
}

type TokenSend struct {
	Id          string
	Amount      uint64
	Destination string
}

func (l *TreeLedger) SendToken(txId, destination string, amount uint64) (*dag.Dag, error) {
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

	newSend, err := l.tree.CreateNode(context.TODO(), &TokenSend{
		Id:          txId,
		Amount:      amount,
		Destination: destination,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create new node: %v", err)
	}

	sentCids, err := l.tokenTransactionCidsForType(TokenSendLabel)
	if err != nil {
		return nil, fmt.Errorf("error getting existing token sends: %v", err)
	}

	sentCids = append(sentCids, newSend.Cid())

	newTree, err := l.tree.SetAsLink(context.TODO(), append(tokenPath, TokenSendLabel), sentCids)
	if err != nil {
		return nil, fmt.Errorf("error setting: %v", err)
	}

	newBalance := availableBalance - amount
	newTree, err = newTree.Set(context.TODO(), append(tokenPath, TokenBalanceLabel), newBalance)
	if err != nil {
		return nil, fmt.Errorf("error updating balance: %v", err)
	}

	l.tree = newTree

	return newTree, nil
}

type TokenReceive struct {
	SendTokenTransactionId string
	Amount                 uint64
}

func (l *TreeLedger) ReceiveToken(sendTokenTxId string, amount uint64) (*dag.Dag, error) {
	tokenPath, err := l.tokenPath()
	if err != nil {
		return nil, err
	}

	tokenExists, err := l.TokenExists()
	if err != nil {
		return nil, err
	}

	if !tokenExists {
		newTree, err := l.createToken()
		if err != nil {
			return nil, err
		}

		l.tree = newTree
	}

	tokenReceives, err := l.tokenTransactionCidsForType(TokenReceiveLabel)
	if err != nil {
		return nil, fmt.Errorf("error getting existing token receives: %v", err)
	}

	// TODO: Consider storing receives as map w/ send tx id keys instead of iterating over all of them
	for _, r := range tokenReceives {
		node, err := l.tree.Get(context.TODO(), r)
		if err != nil {
			return nil, fmt.Errorf("error getting existing token receive: %v", err)
		}

		tr := TokenReceive{}
		err = cbornode.DecodeInto(node.RawData(), &tr)
		if err != nil {
			return nil, fmt.Errorf("error decoding token receive node: %v", err)
		}

		if tr.SendTokenTransactionId == sendTokenTxId {
			return nil, fmt.Errorf("cannot receive token; transaction id %s already exists", tr.SendTokenTransactionId)
		}
	}

	newReceive, err := l.tree.CreateNode(context.TODO(), TokenReceive{
		SendTokenTransactionId: sendTokenTxId,
		Amount:                 amount,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating token receive node: %v", err)
	}

	tokenReceives = append(tokenReceives, newReceive.Cid())

	newTree, err := l.tree.SetAsLink(context.TODO(), append(tokenPath, TokenReceiveLabel), tokenReceives)
	if err != nil {
		return nil, fmt.Errorf("error setting: %v", err)
	}

	currBalance, err := l.Balance()
	if err != nil {
		return nil, fmt.Errorf("error getting current balance: %v", err)
	}

	newBalance := currBalance + amount
	newTree, err = newTree.Set(context.TODO(), append(tokenPath, TokenBalanceLabel), newBalance)
	if err != nil {
		return nil, fmt.Errorf("error updating balance: %v", err)
	}

	l.tree = newTree

	return newTree, nil
}
