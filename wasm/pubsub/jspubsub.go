// +build wasm

package pubsub

import (
	"fmt"
	"syscall/js"

	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/helpers"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// PubSubBridge is a bridge where golang (in wasm) can use the underlying javascript pubsub
// for (for example) the tupelo client
type PubSubBridge struct {
	remote.UnderlyingPubSub
	jspubsub js.Value
}

func NewPubSubBridge(jspubsub js.Value) *PubSubBridge {
	return &PubSubBridge{
		jspubsub: jspubsub,
	}
}

func (psb *PubSubBridge) Publish(topic string, data []byte) error {
	resp := make(chan error)
	defer close(resp)
	psb.jspubsub.Call("publish", js.ValueOf(topic), helpers.SliceToJSBuffer(data), js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
		go func() {
			if len(args) > 0 && args[0].Truthy() {
				resp <- fmt.Errorf("error publishing: %s", args[0].String())
				return
			}
			resp <- nil
		}()

		return nil
	}))

	return <-resp
}

func (psb *PubSubBridge) Subscribe(topic string, opts ...pubsub.SubOpt) (remote.UnderlyingSubscription, error) {
	sub := newBridgedSubscription(topic)
	resp := make(chan error)
	doneFunc := js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			resp <- nil
			return nil
		}
		resp <- fmt.Errorf("error subscribing: %s", args[0].String())
		return nil
	})
	go func() {
		psb.jspubsub.Call("subscribe", js.ValueOf(topic), js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
			go func() {
				sub.QueueJS(args[0])
			}()
			return nil
		}), doneFunc)
	}()
	err := <-resp
	doneFunc.Release()
	if err != nil {
		return nil, err
	}
	return sub, nil
}
