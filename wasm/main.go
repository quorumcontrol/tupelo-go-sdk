// +build wasm

package main

import (
	"time"
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


			jsObj.Set("testpubsub", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				t := then.New()
				go func() {
					gopub := pubsub.NewPubSubBridge(args[0])
					sub,err := gopub.Subscribe("test")
					if err != nil {
						t.Reject(err.Error())
					} 
					ctx,cancel := context.WithTimeout(context.Background(), 2 * time.Second)
					defer cancel()
					
					err = gopub.Publish("test", []byte("hi"))
					if err != nil {
						t.Reject(err.Error())
					} 
					
					msg,err := sub.Next(ctx)
					if err == nil {
						t.Resolve(js.TypedArrayOf(msg.Data))
					} else {
						t.Reject(err.Error())
					}
				}()
				return t
			}))
			return jsObj
		}),
	)

	js.Global().Get("Go").Call("readyResolver")

	<-exitChan
}
