package remote

import (
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-client/p2p"
)

type actorRegistry map[string]*actor.PID

type remoteManager struct {
	gateways actorRegistry
}

// These are GLOBAL state used to handle the singleton for routing to remote hosts
// and the registry of local bridges
var globalManager *remoteManager

// Start starts the remote management.
func Start() {
	if globalManager == nil {
		globalManager = newRemoteManager()
		actor.ProcessRegistry.RegisterAddressResolver(remoteHandler)
	}
}

// Stop stops the remote management.
func Stop() {
	globalManager.stop()
	globalManager = nil
}

// NewRouter instantiates a new router given a certain host node.
func NewRouter(ctx *actor.RootContext, host p2p.Node) *actor.PID {
	middleware.Log.Infow("registering router", "host", host.Identity())
	router, err := ctx.SpawnNamed(newRouterProps(host), "router-"+host.Identity())
	if err != nil {
		panic(fmt.Sprintf("error spawning router: %v", err))
	}
	globalManager.gateways[host.Identity()] = router
	return router
}

func remoteHandler(pid *actor.PID) (actor.Process, bool) {
	from := types.RoutableAddress(pid.Address).From()
	for gateway, router := range globalManager.gateways {
		if from == gateway {
			ref := newProcess(pid, router)
			return ref, true
		}
	}
	middleware.Log.Errorw("unhandled remote pid", "addr", pid.Address, "current", globalManager.gateways)
	panic(fmt.Sprintf("unhandled remote pid: %s id: %s", pid.Address, pid.GetId()))
}

func newRemoteManager() *remoteManager {
	rm := &remoteManager{
		gateways: make(actorRegistry),
	}

	return rm
}

func (rm *remoteManager) stop() {
	for _, router := range rm.gateways {
		router.GracefulStop()
	}
}
