package remote

import (
	"fmt"
	"strings"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/opentracing/opentracing-go"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/quorumcontrol/tupelo-go-sdk/tracing"
	// "github.com/quorumcontrol/tupelo-go-sdk/tracing"
)

const p2pProtocol = "remoteactors/v1.0"

type router struct {
	middleware.LogAwareHolder

	bridges actorRegistry
	host    p2p.Node
}

func newRouterProps(host p2p.Node) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &router{
			host:    host,
			bridges: make(actorRegistry),
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (r *router) Receive(context actor.Context) {
	// defer func() {
	// 	if re := recover(); re != nil {
	// 		r.Log.Errorw("recover", "re", re)
	// 	}
	// }()
	switch msg := context.Message().(type) {
	case *actor.Terminated:
		split := strings.Split(msg.Who.GetId(), "/")
		to := split[len(split)-1]
		if _, ok := r.bridges[to]; ok {
			delete(r.bridges, to)
			return
		}
		r.Log.Errorw("unknown actor died", "who", msg.Who)
	case *actor.Started:
		// don't keep a reference to the context around,
		// instead create a closure which has just the PID
		// of the router and uses the root context to send the streams
		routerActor := context.Self()
		r.host.SetStreamHandler(p2pProtocol, func(s network.Stream) {
			actor.EmptyRootContext.Send(routerActor, s)
		})
	case network.Stream:
		remoteGateway := msg.Conn().RemotePeer().Pretty()
		handler, ok := r.bridges[remoteGateway]
		if !ok {
			handler = r.createBridge(context, remoteGateway)
		}
		context.Forward(handler)
	case *wireDeliveryWrapper:
		var sp opentracing.Span

		if traceable, ok := msg.originalMessage.(tracing.Traceable); ok && tracing.Enabled && msg.Outgoing && traceable.Started() {
			sp = traceable.NewSpan("router-outgoing")
			defer sp.Finish()
		}

		target := types.RoutableAddress(msg.Target.Address)
		if sp != nil {
			sp.SetTag("target", target.To())
		}
		handler, ok := r.bridges[target.To()]
		if sp != nil && ok {
			sp.SetTag("existing-bridge", handler.String())
		}
		if !ok {
			handler = r.createBridge(context, target.To())
			if sp != nil {
				sp.SetTag("new-bridge", handler.String())
			}
		}
		context.Forward(handler)
	}
}

func (r *router) createBridge(context actor.Context, to string) *actor.PID {
	p, err := peer.IDB58Decode(to)
	if err != nil {
		panic(fmt.Sprintf("error decoding pretty: %s", to))
	}
	bridge, err := context.SpawnNamed(newBridgeProps(r.host, p), p.Pretty())
	if err != nil {
		panic(fmt.Sprintf("error spawning bridge: %v", err))
	}
	r.bridges[p.Pretty()] = bridge
	return bridge
}