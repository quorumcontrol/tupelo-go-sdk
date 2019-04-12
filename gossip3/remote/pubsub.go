package remote

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/p2p"
	"github.com/quorumcontrol/tupelo-go-client/tracing"
)

const topicNamePrefix = "tupelo-broadcast"

// NetworkBroadcaster implements the broadcast interface necessary
// for the client
type NetworkBroadcaster struct {
	host p2p.Node
}

// NewNetworkBroadcaster returns a NetworkBroadcaster that can be used
// to send messages.WireMessage across the p2p network using pubsub.
func NewNetworkBroadcaster(host p2p.Node) *NetworkBroadcaster {
	return &NetworkBroadcaster{
		host: host,
	}
}

// Broadcast sends the message over the wire to any receivers
func (nb *NetworkBroadcaster) Broadcast(message messages.WireMessage) error {
	msg, ok := message.(messages.WireMessage)
	if !ok {
		return fmt.Errorf("error, message of type %s is not a messages.WireMessage", reflect.TypeOf(msg).String())
	}
	marshaled, err := msg.MarshalMsg(nil)
	if err != nil {
		return fmt.Errorf("could not marshal message: %v", err)
	}

	wd := &WireDelivery{
		originalMessage: msg,
		Message:         marshaled,
		Type:            msg.TypeCode(),
		Target:          nil, // specifically nil because it's broadcast
		Sender:          nil, // specifically nil because there is no response possible on broadcast
	}
	bits, err := wd.MarshalMsg(nil)
	if err != nil {
		return fmt.Errorf("error marshaling message: %v", err)
	}

	return nb.host.Publish(topicNameFromTypeCode(wd.Type), bits)
}

func topicNameFromTypeCode(typeCode int8) string {
	return topicNamePrefix + "-" + strconv.Itoa(int(typeCode))
}

type broadcastSubscriber struct {
	middleware.LogAwareHolder
	tracing.ContextHolder

	subCtx     context.Context
	cancelFunc context.CancelFunc
	host       p2p.Node
	topicName  string
}

// A NetworkSubscriber is a subscription to a pubsub style system for a specific message type
// it is designed to be spawned inside another context so that it can use Parent in order to
// deliver the messages
func NewNetworkSubscriberProps(typeCode int8, host p2p.Node) *actor.Props {
	ctx, cancel := context.WithCancel(context.Background())
	return actor.PropsFromProducer(func() actor.Actor {
		return &broadcastSubscriber{
			host:       host,
			cancelFunc: cancel,
			subCtx:     ctx,
			topicName:  topicNameFromTypeCode(typeCode),
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (bs *broadcastSubscriber) Receive(actorContext actor.Context) {
	switch msg := actorContext.Message().(type) {
	case *actor.Started:
		bs.Log.Debugw("subscribed")
		sub, err := bs.host.Subscribe(bs.topicName)
		if err != nil {
			panic(fmt.Sprintf("subscription failed, dying %v", err))
		}
		self := actorContext.Self()
		go func() {
			for {
				bs.Log.Debugw("for loop")
				msg, err := sub.Next(bs.subCtx)
				if err == nil {
					bs.Log.Debugw("msg received", "type", reflect.TypeOf(msg).String())
					actor.EmptyRootContext.Send(self, msg)
				} else {
					if err.Error() == "context canceled" {
						return // end the loop on a context cancel
					}

					bs.Log.Errorw("error getting message", "err", err)
					panic("error getting message")

				}
			}
		}()
	case *actor.Stopping:
		bs.cancelFunc()
	case *pubsub.Message:
		bs.handlePubSubMessage(actorContext, msg)
	}
}

func (bs *broadcastSubscriber) handlePubSubMessage(actorContext actor.Context, pubsubMsg *pubsub.Message) {
	bs.Log.Debugw("received")
	wd := WireDelivery{}
	_, err := wd.UnmarshalMsg(pubsubMsg.Data)
	if err != nil {
		bs.Log.Errorw("error unmarshaling", "err", err)
		return
	}
	msg, err := wd.GetMessage()
	if err != nil {
		bs.Log.Errorw("error getting message", "err", err)
		return
	}
	actorContext.Send(actorContext.Parent(), msg)
}
