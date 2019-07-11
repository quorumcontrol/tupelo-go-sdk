// +build wasm

package pubsub

import (
	"context"
	"fmt"
	"syscall/js"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type underlyingSub interface {
	Next(context.Context) (*pubsub.Message, error)
	Cancel()
}

type underlyingPubSub interface {
	Publish(topic string, data []byte) error
	RegisterTopicValidator(topic string, fn pubsub.Validator, opts ...pubsub.ValidatorOpt) error
	UnregisterTopicValidator(topic string) error
	Subscribe(topic string, opts ...pubsub.SubOpt) (underlyingSub, error)
}

// PubSubBridge is a bridge where golang (in wasm) can use the underlying javascript pubsub
// for (for example) the tupelo client
type PubSubBridge struct {
	underlyingPubSub
	jspubsub js.Value
}

func NewPubSubBridge(jspubsub js.Value) *PubSubBridge {
	return &PubSubBridge{
		jspubsub: jspubsub,
	}
}

func (psb *PubSubBridge) Publish(topic string, data []byte) error {
	resp := make(chan error)
	psb.jspubsub.Call("publish", js.ValueOf(topic), js.TypedArrayOf(data), js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
		go func() {
			fmt.Println("publish callback called")
			if len(args) > 0 && args[0].Truthy() {
				resp <- fmt.Errorf("error publishing")
				return
			}
			resp <- nil
		}()

		return nil
	}))

	return <-resp
}
