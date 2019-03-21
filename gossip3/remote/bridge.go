package remote

import (
	gocontext "context"
	"fmt"
	"math/rand"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	pnet "github.com/ipsn/go-ipfs/gxlibs/github.com/libp2p/go-libp2p-net"
	peer "github.com/ipsn/go-ipfs/gxlibs/github.com/libp2p/go-libp2p-peer"
	"github.com/opentracing/opentracing-go"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-client/p2p"
	"github.com/quorumcontrol/tupelo-go-client/tracing"
	"github.com/tinylib/msgp/msgp"
)

const maxBridgeBackoffs = 10

type internalStreamDied struct {
	id  uint64
	err error
}

type bridge struct {
	middleware.LogAwareHolder
	host         p2p.Node
	localAddress string

	remoteAddress peer.ID

	streamID     uint64
	stream       pnet.Stream
	streamCtx    gocontext.Context
	streamCancel gocontext.CancelFunc
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
	// defer func() {
	// 	if re := recover(); re != nil {
	// 		b.Log.Errorw("recover", "re", re)
	// 		panic(re)
	// 	}
	// }()
	switch msg := context.Message().(type) {
	case *actor.ReceiveTimeout:
		b.Log.Infow("terminating stream due to lack of activity")
		b.behavior.Become(b.TerminatedState)
		b.clearStream(b.streamID)
		context.Self().Poison()
	case *actor.Stopped:
		b.clearStream(b.streamID)
	case pnet.Stream:
		context.SetReceiveTimeout(bridgeReceiveTimeout)
		b.handleIncomingStream(context, msg)
	case *internalStreamDied:
		b.handleStreamDied(context, msg)
	case *WireDelivery:
		context.SetReceiveTimeout(bridgeReceiveTimeout)
		if msg.Outgoing {
			b.handleOutgoingWireDelivery(context, msg)
		} else {
			b.handleIncomingWireDelivery(context, msg)
		}
	}
}

func (b *bridge) TerminatedState(context actor.Context) {
	b.Log.Errorw("received message in terminated state", "msg", context.Message())
}

func (b *bridge) handleIncomingStream(context actor.Context, stream pnet.Stream) {
	remote := stream.Conn().RemotePeer()
	b.Log.Debugw("handling incoming stream", "peer", remote)
	if remote != b.remoteAddress {
		b.Log.Errorw("ignoring stream from other peer", "peer", remote)
	}
	b.clearStream(b.streamID)
	ctx, cancel := gocontext.WithCancel(gocontext.Background())
	b.setupNewStream(ctx, cancel, context, stream)
}

func (b *bridge) handleIncomingWireDelivery(context actor.Context, wd *WireDelivery) {
	msg, err := wd.GetMessage()
	if err != nil {
		panic(fmt.Sprintf("error unmarshaling message: %v", err))
	}

	var sp opentracing.Span

	if tracing.Enabled {
		traceable, ok := msg.(tracing.Traceable)
		if ok && wd.Tracing != nil {
			sp, err = traceable.RehydrateSerialized(wd.Tracing, "bridge-incoming")
			if err == nil {
				sp.SetTag("runthrough-"+time.Now().String(), true)
				defer sp.Finish()
			} else {
				middleware.Log.Debugw("error rehydrating", "err", err)
			}
		}
	}

	// b.Log.Debugw("received", "target", wd.Target, "sender", wd.Sender, "msgHash", crypto.Keccak256(wd.Message))
	var sender *actor.PID
	target := messages.FromActorPid(wd.Target)
	if wd.Sender != nil {
		sender = messages.FromActorPid(wd.Sender)
		sender.Address = types.RoutableAddress(sender.Address).Swap().String()
	}
	// switch the target to the local actor system
	target.Address = actor.ProcessRegistry.Address

	dest, ok := msg.(messages.DestinationSettable)
	if ok {
		orig := dest.GetDestination()
		orig.Address = b.localAddress
		dest.SetDestination(orig)
	}
	if sp != nil {
		sp.SetTag("sending-to-target", true)
	}

	context.RequestWithCustomSender(target, msg, sender)
}

func (b *bridge) handleOutgoingWireDelivery(context actor.Context, wd *WireDelivery) {
	var sp opentracing.Span

	if tracing.Enabled && wd.Started() {
		serialized, err := wd.SerializedContext()
		if err == nil {
			wd.Tracing = serialized
		} else {
			b.Log.Errorw("error serializing", "err", err)
		}
		// this is intentionally after the serialized
		// it will complete before the next in the sequence
		// starts and shouldn't be the span that's still
		// open when going across the wire
		sp = wd.NewSpan("bridge-outgoing")
		defer sp.Finish()
	}

	if b.stream == nil {
		if sp != nil {
			sp.SetTag("existing-stream", false)
		}
		err := b.handleCreateNewStream(context)
		if err != nil {
			b.Log.Warnw("error opening stream", "err", err)
			// back off dialing if we have trouble with the stream
			// non-cryptographic random here to add jitter
			time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
			b.backoffCount++
			if b.backoffCount > maxBridgeBackoffs {
				context.Self().Stop()
				b.Log.Errorw("maximum backoff reached - possibly dropped messages", "count", b.backoffCount)
				return
			}
			if sp != nil {
				sp.SetTag("error", true)
				sp.SetTag("error-opening-stream", err)
				wd.NewSpan("resending").Finish()
			}
			context.Forward(context.Self())
			return
		}
		if sp != nil {
			sp.SetTag("new-stream", true)
		}
	}
	// b.Log.Debugw("writing", "target", wd.Target, "sender", wd.Sender, "msgHash", crypto.Keccak256(wd.Message))
	if wd.Sender != nil && wd.Sender.Address == actor.ProcessRegistry.Address {
		wd.Sender.Address = b.localAddress
	}

	var encSpan opentracing.Span
	if sp != nil {
		encSpan = opentracing.StartSpan("bridge-encodeMsg", opentracing.ChildOf(sp.Context()))
	}

	err := msgp.Encode(b.stream, wd)
	if err != nil {
		if sp != nil {
			sp.SetTag("error", true)
			sp.SetTag("error-encoding-message", err)
			wd.NewSpan("resending").Finish()
		}
		b.clearStream(b.streamID)
		context.Forward(context.Self())
		if encSpan != nil {
			encSpan.Finish()
		}
		return
	}

	if encSpan != nil {
		encSpan.Finish()
	}

	b.backoffCount = 0
}

func (b *bridge) handleStreamDied(context actor.Context, msg *internalStreamDied) {
	b.Log.Infow("stream died", "err", msg.err)
	b.clearStream(msg.id)
}

func (b *bridge) clearStream(id uint64) {
	if b.streamID != id {
		b.Log.Errorw("ids did not match", "expecting", id, "actual", b.streamID)
		return
	}
	if b.stream != nil {
		b.stream.Reset()
	}
	if b.streamCancel != nil {
		b.streamCancel()
	}
	b.stream = nil
	b.streamCtx = nil
}

func (b *bridge) handleCreateNewStream(context actor.Context) error {
	if b.stream != nil {
		b.Log.Infow("not creating new stream, already exists")
		return nil
	}
	b.Log.Debugw("create new stream")

	ctx, cancel := gocontext.WithCancel(gocontext.Background())

	stream, err := b.host.NewStreamWithPeerID(ctx, b.remoteAddress, p2pProtocol)
	if err != nil {
		b.clearStream(b.streamID)
		cancel()
		return err
	}

	b.setupNewStream(ctx, cancel, context, stream)
	return nil
}

func (b *bridge) setupNewStream(ctx gocontext.Context, cancelFunc gocontext.CancelFunc, context actor.Context, stream pnet.Stream) {
	b.stream = stream
	b.streamCtx = ctx
	b.streamCancel = cancelFunc
	b.streamID++
	// b.writer = msgp.NewWriter(stream)
	msgChan := make(chan WireDelivery)

	self := context.Self()

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
		done := ctx.Done()
		reader := msgp.NewReader(stream)

		for {
			select {
			case <-done:
				return
			default:
				var wd WireDelivery
				err := wd.DecodeMsg(reader)
				if err != nil {
					actor.EmptyRootContext.Send(self, internalStreamDied{id: b.streamID, err: err})
					return
				}
				msgChan <- wd
			}
		}
	}()

	go func() {
		done := ctx.Done()
		for {
			select {
			case <-done:
				b.Log.Debugw("resetting stream due to done")
				err := stream.Close()
				if err != nil {
					b.Log.Errorw("error closing stream", "err", err)
				}
				return
			case wd := <-msgChan:
				actor.EmptyRootContext.Send(self, &wd)
			}
		}
	}()
}
