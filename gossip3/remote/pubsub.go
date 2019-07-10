package remote

import (
	"context"
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-sdk/tracing"
	"go.uber.org/zap"
)

type PubSubValidator func(context.Context, peer.ID, proto.Message) bool

type PubSub interface {
	Broadcast(topic string, msg proto.Message) error
	NewSubscriberProps(topic string) *actor.Props
	RegisterTopicValidator(topic string, validatorFunc PubSubValidator, opts ...pubsub.ValidatorOpt) error
	UnregisterTopicValidator(topic string)
	Subscribe(ctx spawner, topic string, subscribers ...*actor.PID) *actor.PID
}

type libp2pSub interface {
	Next(context.Context) (*pubsub.Message, error)
	Cancel()
}

type libp2pPubsub interface {
	Publish(topic string, data []byte) error
	RegisterTopicValidator(topic string, fn pubsub.Validator, opts ...pubsub.ValidatorOpt) error
	UnregisterTopicValidator(topic string) error
	Subscribe(topic string, opts ...pubsub.SubOpt) (*pubsub.Subscription, error)
}

type spawner interface {
	Spawn(props *actor.Props) *actor.PID
}

// NetworkPubSub implements the broadcast interface necessary
// for the client
type NetworkPubSub struct {
	pubsub libp2pPubsub
	log    *zap.SugaredLogger
}

// NewNetworkPubSub returns a NetworkBroadcaster that can be used
// to send messages.WireMessage across the p2p network using pubsub.
func NewNetworkPubSub(pubsub libp2pPubsub) *NetworkPubSub {
	return &NetworkPubSub{
		pubsub: pubsub,
		log:    middleware.Log.Named("network-pubsub"),
	}
}

// Broadcast sends the message over the wire to any receivers
func (nps *NetworkPubSub) Broadcast(topic string, message proto.Message) error {
	if traceable, ok := message.(tracing.Traceable); ok {
		sp := traceable.NewSpan("pubsub-publish")
		defer sp.Finish()
	}

	any, err := ptypes.MarshalAny(message)
	if err != nil {
		return fmt.Errorf("could not marshal message to any: %v", err)
	}

	marshaled, err := proto.Marshal(any)
	if err != nil {
		return fmt.Errorf("could not marshal any: %v", err)
	}

	nps.log.Debugw("broadcasting", "topic", topic)
	return nps.pubsub.Publish(topic, marshaled)
}

func (nps *NetworkPubSub) RegisterTopicValidator(topic string, validatorFunc PubSubValidator, opts ...pubsub.ValidatorOpt) error {
	var wrappedFunc pubsub.Validator = func(ctx context.Context, peer peer.ID, pubsubMsg *pubsub.Message) bool {
		msg, err := pubsubMessageToProtoMessage(pubsubMsg)
		if err != nil {
			nps.log.Errorw("error getting wire message", "err", err)
			return false
		}
		if traceable, ok := msg.(tracing.Traceable); ok {
			sp := traceable.NewSpan("validating")
			defer sp.Finish()
		}
		return validatorFunc(ctx, peer, msg)
	}
	return nps.pubsub.RegisterTopicValidator(topic, wrappedFunc, opts...)
}

func (nps *NetworkPubSub) UnregisterTopicValidator(topic string) {
	if err := nps.pubsub.UnregisterTopicValidator(topic); err != nil {
		nps.log.Errorw("error unregistering validator", "err", err)
	}
}

func (nps *NetworkPubSub) NewSubscriberProps(topic string) *actor.Props {
	return newBroadcastSubscriberProps(topic, nps.pubsub, true)
}

func (nps *NetworkPubSub) Subscribe(ctx spawner, topic string, subscribers ...*actor.PID) *actor.PID {
	return ctx.Spawn(newBroadcastSubscriberProps(topic, nps.pubsub, false, subscribers...))
}

type broadcastSubscriber struct {
	middleware.LogAwareHolder
	tracing.ContextHolder

	subCtx       context.Context
	cancelFunc   context.CancelFunc
	subscription libp2pSub
	pubsub       libp2pPubsub
	topicName    string
	subscribers  []*actor.PID
	notifyParent bool
	stopped      bool
}

// A NetworkSubscriber is a subscription to a pubsub style system for a specific message type
// it is designed to be spawned inside another context so that it can use Parent in order to
// deliver the messages
func newBroadcastSubscriberProps(topic string, pubsub libp2pPubsub, notifyParent bool, subscribers ...*actor.PID) *actor.Props {
	ctx, cancel := context.WithCancel(context.Background())
	return actor.PropsFromProducer(func() actor.Actor {
		return &broadcastSubscriber{
			pubsub:       pubsub,
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
		sub, err := bs.pubsub.Subscribe(bs.topicName)
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
	bs.Log.Debugw("received pubsub message", "topic", bs.topicName)
	msg, err := pubsubMessageToProtoMessage(pubsubMsg)
	if err != nil {
		bs.Log.Errorw("error getting wire message", "err", err)
		return
	}

	bs.Log.Debugw("converted to wire message")
	if traceable, ok := msg.(tracing.Traceable); ok {
		sp := traceable.NewSpan("pubsub-receive")
		defer sp.Finish()
	}
	if bs.notifyParent {
		bs.Log.Debugw("notifying parent actor")
		actorContext.Send(actorContext.Parent(), msg)
	}
	for _, subscriber := range bs.subscribers {
		actorContext.Send(subscriber, msg)
	}
}

func pubsubMessageToProtoMessage(pubsubMsg *pubsub.Message) (proto.Message, error) {
	any := &ptypes.Any{}
	err := proto.Unmarshal(pubsubMsg.Data, any)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling: %v", err)
	}

	dn := &ptypes.DynamicAny{}

	err = ptypes.UnmarshalAny(any, dn)
	if err != nil {
		return nil, fmt.Errorf("error getting message: %v", err)
	}
	return dn.Message, nil
}
