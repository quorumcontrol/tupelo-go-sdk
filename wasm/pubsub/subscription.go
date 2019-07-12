// +build wasm

package pubsub

import (
	"context"
	"fmt"
	"syscall/js"

	pb "github.com/libp2p/go-libp2p-pubsub/pb"

	libpubsub "github.com/libp2p/go-libp2p-pubsub"
)

type underlyingSub interface {
	Next(context.Context) (*libpubsub.Message, error)
	Cancel()
}

type BridgedSubscription struct {
	topic string
	ch    chan *libpubsub.Message
}

func newBridgedSubscription(topic string) *BridgedSubscription {
	return &BridgedSubscription{
		topic: topic,
		ch:    make(chan *libpubsub.Message),
	}
}

func (bs *BridgedSubscription) Next(ctx context.Context) (*libpubsub.Message, error) {
	done := ctx.Done()
	select {
	case <-done:
		return nil, fmt.Errorf("context done")
	case msg := <-bs.ch:
		return msg, nil
	}
}

func (bs *BridgedSubscription) Cancel() {
	// do nothing for now
}

func (bs *BridgedSubscription) QueueJS(msg js.Value) {
	fmt.Println("received subscription message")
	// pubsubMsg := &libpubsub.Message{}
	pubsubMsg := &libpubsub.Message{
		Message: &pb.Message{
			From:     []byte(msg.Get("from").String()),
			Data:     jsBufferToByes(msg.Get("data")),
			TopicIDs: []string{msg.Get("topicIDs").Index(0).String()},
		},
	}
	fmt.Println("after assigning")
	bs.ch <- pubsubMsg

	// js looks like:
	// {
	//     from: 'QmSWBdQGuX8Uregx8QSKxCxk1tacQPgJqX1AXnSUnzDyEM',
	//     data: <Buffer 68 69>,
	//     seqno: <Buffer fe 54 4a ad 49 b4 94 5d 09 98 ad 0e ad 70 43 33 4c fc 4f a5>,
	//     topicIDs: [ 'test' ]
	//   }
}

func jsBufferToByes(buf js.Value) []byte {
	len := buf.Length()
	bits := make([]byte, len)
	for i := 0; i < len; i++ {
		bits[i] = byte(uint8(buf.Index(i).Int()))
	}
	return bits
}
