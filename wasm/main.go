// +build wasm

package main

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/quorumcontrol/tupelo-go-sdk/wasm/helpers"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jscommunity"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jscrypto"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/then"

	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jslibs"

	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jsclient"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/pubsub"
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

			jsObj.Set("passPhraseKey", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return jsclient.PassPhraseKey(args[0], args[1])
			}))

			jsObj.Set("keyFromPrivateBytes", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return jsclient.KeyFromPrivateBytes(args[0])
			}))

			jsObj.Set("ecdsaPubkeyToDid", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return jsclient.EcdsaPubkeyToDid(args[0])
			}))

			jsObj.Set("ecdsaPubkeyToAddress", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return jsclient.EcdsaPubkeyToAddress(args[0])
			}))

			jsObj.Set("newEmptyTree", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return jsclient.NewEmptyTree(args[0], args[1])
			}))

			jsObj.Set("getCurrentState", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				jsOpts := args[0]
				return jscommunity.GetCurrentState(context.TODO(), jsOpts.Get("tip"), jsOpts.Get("blockService"), jsOpts.Get("did"))
			}))

			jsObj.Set("tokenPayloadForTransaction", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				jsOpts := args[0]
				return jsclient.TokenPayloadForTransaction(
					jsOpts.Get("blockService"),
					jsOpts.Get("tip"),
					jsOpts.Get("tokenName"),
					jsOpts.Get("sendId"),
					jsOpts.Get("jsSendTxSig"),
				)
			}))

			// hashToShard(topicName:string, shardCount:number):number
			jsObj.Set("hashToShardNumber", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return jscommunity.HashToShardNumber(args[0].String(), args[1].Int())
			}))

			jsObj.Set("sign", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return jscrypto.Sign(helpers.JsBufferToBytes(args[0]), helpers.JsBufferToBytes(args[1]))
			}))

			jsObj.Set("numberToBytes", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return helpers.SliceToJSBuffer(jscommunity.NumberToBytes(uint64(args[0].Int())))
			}))

			jsObj.Set("getSendableEnvelopeBytes", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return jscommunity.GetSendableBytes(helpers.JsBufferToBytes(args[0]), helpers.JsBufferToBytes(args[1]))
			}))

			jsObj.Set("playTransactions", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				// js passes in:
				// interface IPlayTransactionOptions {
				//     publisher: IPubSub,
				//     notaryGroup: Uint8Array // protobuf encoded config.NotaryGroup
				//     blockService: IBlockService,
				//     privateKey: Uint8Array,
				//     tip: CID,
				//     transactions: Uint8Array[], // protobuf encoded array of transactions.Transaction
				// }
				jsOpts := args[0]

				config, err := jsclient.JsConfigToHumanConfig(jsOpts.Get("notaryGroup"))
				if err != nil {
					t := then.New()
					t.Reject(errors.Wrap(err, "error converting config").Error())
					return t
				}

				bridge := pubsub.NewPubSubBridge(jsOpts.Get("publisher"))
				cli := jsclient.New(bridge, config)
				return cli.PlayTransactions(jsOpts.Get("blockService"), jsOpts.Get("privateKey"), jsOpts.Get("tip"), jsOpts.Get("transactions"))
			}))

			return jsObj
		}),
	)

	go js.Global().Get("Go").Call("readyResolver")

	<-exitChan
}
