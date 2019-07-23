// +build wasm

package main

import (
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jslibs"
	"fmt"
	"syscall/js"

	"github.com/quorumcontrol/tupelo-go-sdk/wasm/pubsub"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jsclient"
)

var exitChan chan bool

func init() {
	exitChan = make(chan bool)
}

func main() {
	js.Global().Get("Go").Set("exit", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		exitChan <- true
		return nil
	}))

	js.Global().Set(
		"populateLibrary",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) != 2 || !args[0].Truthy() || !args[1].Truthy() {
				err := fmt.Errorf("error, must supply a valid object")
				panic(err)
			}
			
			helperLibs := args[1]
			cids := helperLibs.Get("cids")
			ipfsBlock := helperLibs.Get("ipfs-block")
			if !cids.Truthy() || !ipfsBlock.Truthy() {
				err := fmt.Errorf("error, must supply a library object containing cids and ipfs-block")
				go fmt.Println(err.Error())
				panic(err)
			}
			jslibs.Cids = cids
			jslibs.IpfsBlock = ipfsBlock

			jsObj := args[0]

			jsObj.Set("generateKey", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return jsclient.GenerateKey()
			}))

			jsObj.Set("newEmptyTree", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return jsclient.NewEmptyTree(args[0], args[1])
			}))

			jsObj.Set("playTransactions",  js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				// js passes in:
				// interface IPlayTransactionOptions {
				//     publisher: IPubSub,
				//     blockService: IBlockService, 
				//     privateKey: Uint8Array,
				//     tip: CID, 
				//     transactions: Uint8Array[],
				// }
				jsOpts := args[0]

				bridge := pubsub.NewPubSubBridge(jsOpts.Get("publisher"))
				cli := jsclient.New(bridge)
				go fmt.Println("play transactionst")
				return cli.PlayTransactions(jsOpts.Get("blockService"), jsOpts.Get("privateKey"), jsOpts.Get("tip"), jsOpts.Get("transactions"))
			}))

			return jsObj
		}),
	)

	js.Global().Get("Go").Call("readyResolver")

	<-exitChan
}
