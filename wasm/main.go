package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func doIt() error {
	ctx := context.TODO()

	key, err := crypto.GenerateKey()
	if err != nil {
		return err
	}

	store := nodestore.MustMemoryStore(ctx)
	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := consensus.NewEmptyTree(treeDID, store)

	tokenName := "testtoken"
	// tokenFullName := strings.Join([]string{treeDID, tokenName}, ":")

	txn, err := chaintree.NewEstablishTokenTransaction(tokenName, 42)
	if err != nil {
		return err
	}

	blockWithHeaders := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	testTree, err := chaintree.NewChainTree(ctx, emptyTree, nil, consensus.DefaultTransactors)
	if err != nil {
		return err
	}

	_, err = testTree.ProcessBlock(ctx, &blockWithHeaders)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	err := doIt()
	if err != nil {
		panic(err)
	}
	fmt.Println("we did all the things")
}
