package remote

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/tracing"
)

func NewSimulatedPubSub() *SimulatedPubSub {
	return &SimulatedPubSub{
		eventStream: &eventstream.EventStream{},
	}
}

// SimulatedBroadcaster is a simulated in-memory pubsub that doesn't need a network connection
type SimulatedPubSub struct {
	eventStream *eventstream.EventStream
}

type simulatorMessage struct {
	topic string
	msg   messages.WireMessage
}

// Implements the broadcast necessary for the client side to send to the network
func (sb *SimulatedPubSub) Broadcast(topic string, message messages.WireMessage) error {
	middleware.Log.Debugw("publishing")
	sb.eventStream.Publish(&simulatorMessage{
		topic: topic,
		msg:   message,
	})
	return nil
}

// returns subscriber props that can be used to listent to broadcast events
func (sb *SimulatedPubSub) NewSubscriberProps(topic string) *actor.Props {
	return newSimulatedSubscriberProps(topic, sb.eventStream, true)
}

func (sb *SimulatedPubSub) Subscribe(ctx spawner, topic string, subscribers ...*actor.PID) *actor.PID {
	return ctx.Spawn(newSimulatedSubscriberProps(topic, sb.eventStream, false, subscribers...))
}

type simulatedSubscriber struct {
	middleware.LogAwareHolder
	tracing.ContextHolder

	subscription *eventstream.Subscription
	eventStream  *eventstream.EventStream
	topic        string

	notifyParent bool
	subscribers  []*actor.PID
}

// A NetworkSubscriber is a subscription to a pubsub style system for a specific message type
// it is designed to be spawned inside another context so that it can use Parent in order to
// deliver the messages
func newSimulatedSubscriberProps(topic string, eventStream *eventstream.EventStream, notifyParent bool, subscribers ...*actor.PID) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &simulatedSubscriber{
			topic:        topic,
			eventStream:  eventStream,
			notifyParent: notifyParent,
			subscribers:  subscribers,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (bs *simulatedSubscriber) Receive(actorContext actor.Context) {
	switch actorContext.Message().(type) {
	case *actor.Started:
		bs.Log.Debugw("subscribed", "topic", bs.topic, "subscribers", bs.subscribers)
		parent := actorContext.Parent()
		sub := bs.eventStream.Subscribe(func(evt interface{}) {
			bs.Log.Debugw("received", "topic", bs.topic, "subscribers", bs.subscribers)
			if bs.notifyParent {
				actor.EmptyRootContext.Send(parent, evt.(*simulatorMessage).msg)
			}
			for _, subscriber := range bs.subscribers {
				actor.EmptyRootContext.Send(subscriber, evt.(*simulatorMessage).msg)
			}
		})
		sub.WithPredicate(func(evt interface{}) bool {
			return evt.(*simulatorMessage).topic == bs.topic
		})
		bs.subscription = sub
	case *actor.Stopping:
		bs.eventStream.Unsubscribe(bs.subscription)
	}
}
