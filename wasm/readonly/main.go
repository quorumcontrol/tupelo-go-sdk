// +build wasm

package main

import (
	"context"
	"fmt"
	"syscall/js"

	cbornode "github.com/ipfs/go-ipld-cbor"

	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/messages/build/go/signatures"

	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jscommunity"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jslibs"
)

var exitChan chan bool

func init() {
	exitChan = make(chan bool)
	typecaster.AddType(signatures.Signature{})
	cbornode.RegisterCborType(signatures.Signature{})
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

			jsObj.Set("getCurrentState", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				jsOpts := args[0]
				return jscommunity.GetCurrentState(context.TODO(), jsOpts.Get("tip"), jsOpts.Get("blockService"), jsOpts.Get("did"))
			}))
			return jsObj
		}),
	)

	go js.Global().Get("Go").Call("readyResolver")

	<-exitChan
}
