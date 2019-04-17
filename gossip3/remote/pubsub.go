package remote

import (
	"context"
	"fmt"
	"reflect"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/p2p"
	"github.com/quorumcontrol/tupelo-go-client/tracing"
)

type PubSub interface {
	Broadcast(topic string, msg messages.WireMessage) error
	// RegisterTopicValidator(topic string, val pubsub.Validator, opts ...pubsub.ValidatorOpt) error
	NewSubscriberProps(topic string) *actor.Props
	Subscribe(ctx spawner, topic string, subscribers ...*actor.PID) *actor.PID
}

type spawner interface {
	Spawn(props *actor.Props) *actor.PID
}

// NetworkPubSub implements the broadcast interface necessary
// for the client
type NetworkPubSub struct {
	host p2p.Node
}

// NewNetworkPubSub returns a NetworkBroadcaster that can be used
// to send messages.WireMessage across the p2p network using pubsub.
func NewNetworkPubSub(host p2p.Node) *NetworkPubSub {
	return &NetworkPubSub{
		host: host,
	}
}

// Broadcast sends the message over the wire to any receivers
func (nps *NetworkPubSub) Broadcast(topic string, message messages.WireMessage) error {
	msg, ok := message.(messages.WireMessage)
	if !ok {
		return fmt.Errorf("error, message of type %s is not a messages.WireMessage", reflect.TypeOf(msg).String())
	}
	if traceable, ok := msg.(tracing.Traceable); ok {
		sp := traceable.NewSpan("pubsub-publish")
		defer sp.Finish()
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

	return nps.host.GetPubSub().Publish(topic, bits)
}

func (nps *NetworkPubSub) NewSubscriberProps(topic string) *actor.Props {
	return newBroadcastSubscriberProps(topic, nps.host, true)
}

func (nps *NetworkPubSub) Subscribe(ctx spawner, topic string, subscribers ...*actor.PID) *actor.PID {
	return ctx.Spawn(newBroadcastSubscriberProps(topic, nps.host, false, subscribers...))
}

type broadcastSubscriber struct {
	middleware.LogAwareHolder
	tracing.ContextHolder

	subCtx       context.Context
	cancelFunc   context.CancelFunc
	subscription *pubsub.Subscription
	host         p2p.Node
	topicName    string
	subscribers  []*actor.PID
	notifyParent bool
	stopped      bool
}

// A NetworkSubscriber is a subscription to a pubsub style system for a specific message type
// it is designed to be spawned inside another context so that it can use Parent in order to
// deliver the messages
func newBroadcastSubscriberProps(topic string, host p2p.Node, notifyParent bool, subscribers ...*actor.PID) *actor.Props {
	ctx, cancel := context.WithCancel(context.Background())
	return actor.PropsFromProducer(func() actor.Actor {
		return &broadcastSubscriber{
			host:         host,
			cancelFunc:   cancel,
			subCtx:       ctx,
			topicName:    topic,
			notifyParent: notifyParent,
			subscribers:  subscribers,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (bs *broadcastSubscriber) Receive(actorContext actor.Context) {
	switch msg := actorContext.Message().(type) {
	case *actor.Started:
		bs.Log.Debugw("subscribed", "topic", bs.topicName)
		sub, err := bs.host.GetPubSub().Subscribe(bs.topicName)
		if err != nil {
			panic(fmt.Sprintf("subscription failed, dying %v", err))
		}
		bs.subscription = sub
		self := actorContext.Self()
		go func() {
			for {
				msg, err := sub.Next(bs.subCtx)
				if bs.stopped {
					return // no need to process here anymore
				}
				if err == nil {
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
		bs.stopped = true
		bs.subscription.Cancel()
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
	if traceable, ok := msg.(tracing.Traceable); ok {
		sp := traceable.NewSpan("pubsub-receive")
		defer sp.Finish()
	}
	if bs.notifyParent {
		actorContext.Send(actorContext.Parent(), msg)
	}
	for _, subscriber := range bs.subscribers {
		actorContext.Send(subscriber, msg)
	}
}
