package remote

import (
	gocontext "context"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	mbridge "github.com/quorumcontrol/messages/build/go/bridge"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/gogo/protobuf/io"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/opentracing/opentracing-go"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/quorumcontrol/tupelo-go-sdk/tracing"
)

const maxBridgeBackoffs = 10

type internalStreamDied struct {
	stream *bridgeStream
}

type bridgeStream struct {
	middleware.LogAwareHolder

	host   p2p.Node
	stream network.Stream
	act    *actor.PID
	ctx    gocontext.Context
	cancel gocontext.CancelFunc
}

func (bs *bridgeStream) Stop() error {
	if bs.cancel != nil {
		bs.cancel()
	}
	return nil
}

func (bs *bridgeStream) SetupOutgoing(remoteAddress peer.ID) error {
	stream, err := bs.host.NewStreamWithPeerID(bs.ctx, remoteAddress, p2pProtocol)
	if err != nil {
		return fmt.Errorf("error creating new stream: %v", err)
	}
	bs.stream = stream
	return nil
}

func (bs *bridgeStream) HandleIncoming(s network.Stream) {
	msgChan := make(chan *mbridge.WireDelivery)
	bs.stream = s

	// there is a separate go routine handling message reading/delivery
	// because the DecodeMsg blocks until a message is received
	// or the stream fails (causing an error)
	// previously this was all done in a single for/select loop
	// however the code actually blocks in the `default:` section
	// and so would ignore the <-done until an error came in
	// now this go routine handling message delivery will do the same thing,
	// but will allow the loop below to actually hit the <-done and close the stream
	// which will trigger this go routine to get an error in the DecodeMsg
	// and allow everything to unblock.
	go func() {
		done := bs.ctx.Done()
		reader := io.NewDelimitedReader(bs.stream, 1<<22) // max size of 4MB

		for {
			select {
			case <-done:
				return
			default:
				wd := &mbridge.WireDelivery{}
				err := reader.ReadMsg(wd)
				if err != nil {
					bs.Log.Errorw("error reading", "err", err)
					actor.EmptyRootContext.Send(bs.act, internalStreamDied{stream: bs})
					return
				}
				bs.Log.Debugw("received", "wiredelivery", wd)
				msgChan <- wd
			}
		}
	}()

	go func() {
		done := bs.ctx.Done()
		for {
			select {
			case <-done:
				bs.Log.Debugw("resetting stream due to done")
				if err := bs.stream.Close(); err != nil {
					err := bs.stream.Reset()
					if err != nil {
						bs.Log.Errorw("error closing stream", "err", err)
					}
				}

				return
			case wd := <-msgChan:
				wd.Outgoing = false
				bs.Log.Debugw("forwarding", "wiredelivery", wd)
				actor.EmptyRootContext.Send(bs.act, &wireDeliveryWrapper{WireDelivery: wd})
			}
		}
	}()
}

func newBridgeStream(ctx gocontext.Context, b *bridge, act *actor.PID) *bridgeStream {
	if ctx == nil {
		ctx = gocontext.Background()
	}
	ctx, cancel := gocontext.WithCancel(ctx)
	return &bridgeStream{
		LogAwareHolder: middleware.LogAwareHolder{
			Log: b.Log.Named("bridge-stream"),
		},
		host:   b.host,
		act:    act,
		ctx:    ctx,
		cancel: cancel,
	}
}

type bridge struct {
	middleware.LogAwareHolder
	host         p2p.Node
	localAddress string

	remoteAddress peer.ID

	incomingStream *bridgeStream
	outgoingStream *bridgeStream
	writer         io.WriteCloser

	backoffCount int

	behavior actor.Behavior
}

func newBridgeProps(host p2p.Node, remoteAddress peer.ID) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		b := &bridge{
			host:          host,
			localAddress:  types.NewRoutableAddress(host.Identity(), remoteAddress.Pretty()).String(),
			remoteAddress: remoteAddress,
			behavior:      actor.NewBehavior(),
		}
		b.behavior.Become(b.NormalState)
		return b
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

var bridgeReceiveTimeout = 60 * time.Second

func (b *bridge) Receive(context actor.Context) {
	b.behavior.Receive(context)
}

func (b *bridge) NormalState(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.ReceiveTimeout:
		b.Log.Infow("terminating stream due to lack of activity")
		b.behavior.Become(b.TerminatedState)
		b.clearStreams()
		context.Self().Poison()
	case *actor.Stopped:
		b.clearStreams()
	case network.Stream:
		context.SetReceiveTimeout(bridgeReceiveTimeout)
		b.handleIncomingStream(context, msg)
	case *internalStreamDied:
		b.handleStreamDied(context, msg)
	case *wireDeliveryWrapper:
		context.SetReceiveTimeout(bridgeReceiveTimeout)
		if msg.Outgoing {
			b.handleOutgoingWireDelivery(context, msg)
		} else {
			b.handleIncomingWireDelivery(context, msg)
		}
	default:
		b.Log.Debugw("unknown message received", "type", reflect.TypeOf(msg).String())
	}
}

func (b *bridge) TerminatedState(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Stopping:
		// do nothing
	case *actor.Stopped:
		b.clearStreams()
	default:
		b.Log.Errorw("received message in terminated state", "type", reflect.TypeOf(msg).String(), "msg", msg)
	}
}

func (b *bridge) handleIncomingStream(context actor.Context, stream network.Stream) {
	remote := stream.Conn().RemotePeer()
	b.Log.Debugw("handling incoming stream", "peer", remote)
	if remote != b.remoteAddress {
		b.Log.Errorw("ignoring stream from other peer", "peer", remote)
	}
	if b.incomingStream != nil {
		err := b.incomingStream.Stop()
		if err != nil {
			b.Log.Errorw("error stopping incoming stream", "err", err)
		}
	}
	b.incomingStream = newBridgeStream(gocontext.TODO(), b, context.Self())
	b.incomingStream.HandleIncoming(stream)
}

func (b *bridge) handleIncomingWireDelivery(context actor.Context, wd *wireDeliveryWrapper) {
	dn := &ptypes.DynamicAny{}
	err := ptypes.UnmarshalAny(wd.GetMessage(), dn)
	if err != nil {
		panic(fmt.Sprintf("error unmarshaling message: %v", err))
	}

	msg := dn.Message
	var sp opentracing.Span

	if tracing.Enabled {
		traceable, ok := msg.(tracing.Traceable)
		if ok && wd.Tracing != nil {
			sp, err = traceable.RehydrateSerialized(wd.Tracing, "bridge-incoming")
			if err == nil {
				defer sp.Finish()
			} else {
				middleware.Log.Debugw("error rehydrating", "err", err)
			}
		}
	}

	var sender *actor.PID
	target := fromActorPid(wd.Target)
	if wd.Sender != nil {
		sender = fromActorPid(wd.Sender)
		sender.Address = types.RoutableAddress(sender.Address).Swap().String()
	}
	target.Address = actor.ProcessRegistry.Address

	if sp != nil {
		sp.SetTag("sending-to-target", true)
	}

	context.RequestWithCustomSender(target, msg, sender)
}

func (b *bridge) handleOutgoingWireDelivery(context actor.Context, wd *wireDeliveryWrapper) {
	var sp opentracing.Span

	if traceable, ok := wd.originalMessage.(tracing.Traceable); tracing.Enabled && ok && traceable.Started() {
		serialized, err := traceable.SerializedContext()
		if err == nil {
			wd.Tracing = serialized
		} else {
			b.Log.Errorw("error serializing", "err", err)
		}
		// this is intentionally after the serialized
		// it will complete before the next in the sequence
		// starts and shouldn't be the span that's still
		// open when going across the wire
		sp = traceable.NewSpan("bridge-outgoing")
	} else {
		sp = opentracing.StartSpan("outgoing-untraceable")
	}
	defer sp.Finish()

	if b.outgoingStream == nil {
		sp.SetTag("existing-stream", false)
		b.Log.Debugw("creating new stream from write operation", "remoteAddress", b.remoteAddress)
		bs := newBridgeStream(gocontext.Background(), b, context.Self())
		err := bs.SetupOutgoing(b.remoteAddress)
		if err != nil {
			// back off dialing if we have trouble with the stream
			// non-cryptographic random here to add jitter
			backoff := time.Duration(rand.Intn(500))
			b.Log.Warnw("error opening stream, backing off", "err", err,
				"remoteAddress", b.remoteAddress, "backoff", backoff, "backoffCount", b.backoffCount)
			time.Sleep(backoff * time.Millisecond)
			b.backoffCount++
			if b.backoffCount > maxBridgeBackoffs {
				context.Self().Stop()
				b.Log.Errorw("maximum backoff reached - possibly dropped messages", "count", b.backoffCount)
				return
			}
			sp.SetTag("error", true)
			sp.SetTag("error-opening-stream", err)

			context.Forward(context.Self())
			return
		}
		sp.SetTag("new-stream", true)
		b.writer = io.NewDelimitedWriter(bs.stream)
		b.outgoingStream = bs
	}

	if wd.Sender != nil && wd.Sender.Address == actor.ProcessRegistry.Address {
		wd.Sender.Address = b.localAddress
	}

	encSpan := opentracing.StartSpan("bridge-encodeMsg", opentracing.ChildOf(sp.Context()))
	defer encSpan.Finish()
	err := b.writer.WriteMsg(wd)
	if err != nil {
		sp.SetTag("error", true)
		sp.SetTag("error-encoding-message", err)

		if err = b.outgoingStream.Stop(); err != nil {
			b.Log.Warnw("failed to stop outgoing stream", "err", err)
		}
		b.outgoingStream = nil
		context.Forward(context.Self())

		return
	}

	b.backoffCount = 0
}

func (b *bridge) handleStreamDied(context actor.Context, msg *internalStreamDied) {
	if err := msg.stream.Stop(); err != nil {
		b.Log.Warnw("failed to stop stream", "err", err)
	}
	b.incomingStream = nil
}

func (b *bridge) clearStreams() {
	if b.outgoingStream != nil {
		if err := b.outgoingStream.Stop(); err != nil {
			b.Log.Warnw("failed to stop outgoing stream", "err", err)
		}
		b.outgoingStream = nil
		b.writer = nil
	}
	if b.incomingStream != nil {
		if err := b.incomingStream.Stop(); err != nil {
			b.Log.Warnw("failed to stop incoming stream", "err", err)
		}
		b.incomingStream = nil
	}

}
