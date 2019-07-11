// +build wasm

package pubsub

import (
	"context"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type underlyingSub interface {
	Next(context.Context) (*pubsub.Message, error)
	Cancel()
}

type BridgedSubscription struct {
	ch chan *pubsub.Message
}
