package remote

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/tracing"
)

func NewSimulatedBroadcaster() *SimulatedBroadcaster {
	return &SimulatedBroadcaster{
		eventStream: &eventstream.EventStream{},
	}
}

// SimulatedBroadcaster is a simulated in-memory pubsub that doesn't need a network connection
type SimulatedBroadcaster struct {
	eventStream *eventstream.EventStream
}

// Implements the broadcast necessary for the client side to send to the network
func (sb *SimulatedBroadcaster) Broadcast(message messages.WireMessage) error {
	middleware.Log.Debugw("publishing")
	sb.eventStream.Publish(message)
	return nil
}

// returns subscriber props that can be used to listent to broadcast events
func (sb *SimulatedBroadcaster) NewSubscriberProps(typeCode int8) *actor.Props {
	return newSimulatedSubscriberProps(typeCode, sb.eventStream)
}

type simulatedSubscriber struct {
	middleware.LogAwareHolder
	tracing.ContextHolder

	subscription *eventstream.Subscription
	eventStream  *eventstream.EventStream
	typeCode     int8
}

// A NetworkSubscriber is a subscription to a pubsub style system for a specific message type
// it is designed to be spawned inside another context so that it can use Parent in order to
// deliver the messages
func newSimulatedSubscriberProps(typeCode int8, eventStream *eventstream.EventStream) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &simulatedSubscriber{
			typeCode:    typeCode,
			eventStream: eventStream,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (bs *simulatedSubscriber) Receive(actorContext actor.Context) {
	switch actorContext.Message().(type) {
	case *actor.Started:
		bs.Log.Debugw("subscribed")
		parent := actorContext.Parent()
		sub := bs.eventStream.Subscribe(func(evt interface{}) {
			actor.EmptyRootContext.Send(parent, evt)
		})
		sub.WithPredicate(func(evt interface{}) bool {
			return evt.(messages.WireMessage).TypeCode() == bs.typeCode
		})
		bs.subscription = sub
	case *actor.Stopping:
		bs.eventStream.Unsubscribe(bs.subscription)
	}
}
