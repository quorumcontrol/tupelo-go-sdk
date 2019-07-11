// +build wasm

package main

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/pubsub"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/then"
)

var exitChan chan bool

func init() {
	exitChan = make(chan bool)
}

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
	// err := doIt()
	// if err != nil {
	// 	panic(err)
	// }

	js.Global().Get("Go").Set("exit", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		exitChan <- true
		return nil
	}))

	js.Global().Set(
		"populateLibrary",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) == 0 || !args[0].Truthy() {
				return js.ValueOf(fmt.Errorf("error, must supply a valid object"))
			}

			jsObj := args[0]


			jsObj.Set("publish", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				t := then.New()
				go func() {
					fmt.Println("publish: ", t)
					fmt.Println("args: ", args)
					gopub := pubsub.NewPubSubBridge(args[0])
					err := gopub.Publish("test", []byte("hi"))
					if err == nil {
						t.Resolve(true)
					} else {
						t.Reject(err.Error)
					}
				}()
				return t.ToJS()
			}))
			return jsObj
		}),
	)

	fmt.Println("we did all the things")
	js.Global().Get("Go").Call("readyResolver")

	<-exitChan
}
