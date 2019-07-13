// +build wasm

package main

import (
	"time"
	"context"
	"fmt"
	"syscall/js"

	"github.com/quorumcontrol/tupelo-go-sdk/wasm/pubsub"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/then"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jsclient"
)

var exitChan chan bool

func init() {
	exitChan = make(chan bool)
}

func testPubSub(jsPubSubLibrary js.Value) *then.Then {
	t := then.New()
	go func() {
		gopub := pubsub.NewPubSubBridge(jsPubSubLibrary)
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
}

func testClient(args []js.Value) interface {} {
	go fmt.Println("testclient called")
	bridge := pubsub.NewPubSubBridge(args[0])
	cli := jsclient.New(bridge)
	return cli.PlayTransactions(args[1], args[2])
}

func main() {
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

			jsObj.Set("generateKey", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return jsclient.GenerateKey()
			}))

			jsObj.Set("testclient", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return testClient(args)
			}))

			jsObj.Set("testpubsub", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				return testPubSub(args[0])
			}))
			return jsObj
		}),
	)

	js.Global().Get("Go").Call("readyResolver")

	<-exitChan
}
