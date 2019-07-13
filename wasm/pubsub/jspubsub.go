// +build wasm

package pubsub

import (
	"fmt"
	"syscall/js"

	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"

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
	fmt.Println("publishing")
	resp := make(chan error)
	psb.jspubsub.Call("publish", js.ValueOf(topic), js.Global().Get("Buffer").Call("from", js.TypedArrayOf(data)), js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
		go func() {
			fmt.Println("publish callback called")
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
	go func() {
		psb.jspubsub.Call("subscribe", js.ValueOf(topic), js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
			go func() {
				sub.QueueJS(args[0])
			}()
			return nil
		}), js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
			if len(args) == 0 {
				resp <- nil
				return nil
			}
			resp <- fmt.Errorf("error subscribing: %s", args[0].String())
			return nil
		}))
	}()
	err := <-resp
	if err != nil {
		return nil, err
	}
	return sub, nil
}
