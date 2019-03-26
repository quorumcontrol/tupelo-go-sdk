package remote

import (
	"context"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-client/p2p"
	"github.com/quorumcontrol/tupelo-go-client/testnotarygroup"
	"github.com/quorumcontrol/tupelo-go-client/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBootstrapHost(ctx context.Context, t *testing.T) p2p.Node {
	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	host, err := p2p.NewLibP2PHost(ctx, key, 0)

	require.Nil(t, err)
	require.NotNil(t, host)

	return host
}

func TestLocalStillWorks(t *testing.T) {
	Start()
	defer Stop()

	rootContext := actor.EmptyRootContext

	localPing := rootContext.Spawn(actor.PropsFromFunc(func(ctx actor.Context) {
		switch msg := ctx.Message().(type) {
		case *messages.Ping:
			ctx.Respond(&messages.Pong{Msg: msg.Msg})
		}
	}))

	resp, err := rootContext.RequestFuture(localPing, &messages.Ping{Msg: "hi"}, 1*time.Second).Result()
	require.Nil(t, err)
	assert.Equal(t, resp.(*messages.Pong).Msg, "hi")
}

func TestRemoteMessageSending(t *testing.T) {
	ts := testnotarygroup.NewTestSet(t, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bootstrap := newBootstrapHost(ctx, t)

	host1, err := p2p.NewLibP2PHost(ctx, ts.EcdsaKeys[0], 0)
	require.Nil(t, err)
	host1.Bootstrap(testnotarygroup.BootstrapAddresses(bootstrap))

	host2, err := p2p.NewLibP2PHost(ctx, ts.EcdsaKeys[1], 0)
	require.Nil(t, err)
	host2.Bootstrap(testnotarygroup.BootstrapAddresses(bootstrap))

	host3, err := p2p.NewLibP2PHost(ctx, ts.EcdsaKeys[2], 0)
	require.Nil(t, err)
	host3.Bootstrap(testnotarygroup.BootstrapAddresses(bootstrap))

	err = host1.WaitForBootstrap(1, 1*time.Second)
	require.Nil(t, err)
	err = host2.WaitForBootstrap(1, 1*time.Second)
	require.Nil(t, err)
	err = host3.WaitForBootstrap(1, 1*time.Second)
	require.Nil(t, err)

	t.Logf("host1: %s / host2: %s / host3: %s", host1.Identity(), host2.Identity(), host3.Identity())

	pingFunc := func(ctx actor.Context) {
		switch msg := ctx.Message().(type) {
		case *messages.Ping:
			// t.Logf("ctx: %v, msg: %v, sender: %v", ctx, msg, ctx.Sender().Address+ctx.Sender().GetId())
			ctx.Respond(&messages.Pong{Msg: msg.Msg})
		}
	}

	rootContext := actor.EmptyRootContext

	host1Ping, err := rootContext.SpawnNamed(actor.PropsFromFunc(pingFunc), "ping-host1")
	require.Nil(t, err)
	defer host1Ping.Poison()

	host2Ping, err := rootContext.SpawnNamed(actor.PropsFromFunc(pingFunc), "ping-host2")
	require.Nil(t, err)
	defer host2Ping.Poison()

	host3Ping, err := rootContext.SpawnNamed(actor.PropsFromFunc(pingFunc), "ping-host3")
	require.Nil(t, err)
	defer host3Ping.Poison()

	Start()
	defer Stop()

	NewRouter(host1)
	NewRouter(host2)
	NewRouter(host3)

	t.Run("ping", func(t *testing.T) {
		remotePing := actor.NewPID(types.NewRoutableAddress(host1.Identity(), host3.Identity()).String(), host3Ping.GetId())

		resp, err := rootContext.RequestFuture(remotePing, &messages.Ping{Msg: "hi"}, 100*time.Millisecond).Result()

		assert.Nil(t, err)
		assert.Equal(t, resp.(*messages.Pong).Msg, "hi")
	})

	t.Run("sending a traceable when tracing is on", func(t *testing.T) {
		tracing.StartJaeger("test-only")
		defer tracing.StopJaeger()
		remotePing := actor.NewPID(types.NewRoutableAddress(host1.Identity(), host3.Identity()).String(), host3Ping.GetId())
		msg := &messages.Ping{Msg: "hi"}
		msg.StartTrace("test-only-ping")
		defer msg.StopTrace()
		resp, err := rootContext.RequestFuture(remotePing, msg, 100*time.Millisecond).Result()

		assert.Nil(t, err)
		assert.Equal(t, resp.(*messages.Pong).Msg, "hi")
	})

	t.Run("when the otherside is closed permanently", func(t *testing.T) {
		newCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		host4, err := p2p.NewLibP2PHost(newCtx, ts.EcdsaKeys[3], 0)
		require.Nil(t, err)
		host4.Bootstrap(testnotarygroup.BootstrapAddresses(bootstrap))
		err = host4.WaitForBootstrap(1, 1*time.Second)
		require.Nil(t, err)
		host4Ping, err := rootContext.SpawnNamed(actor.PropsFromFunc(pingFunc), "ping-host4")
		require.Nil(t, err)
		remote4Ping := actor.NewPID(types.NewRoutableAddress(host1.Identity(), host4.Identity()).String(), host4Ping.GetId())

		NewRouter(host4)

		resp, err := rootContext.RequestFuture(remote4Ping, &messages.Ping{Msg: "hi"}, 100*time.Millisecond).Result()
		assert.Equal(t, resp.(*messages.Pong).Msg, "hi")
		assert.Nil(t, err)

		host4Ping.Stop()
		cancel()

		resp, err = rootContext.RequestFuture(remote4Ping, &messages.Ping{Msg: "hi"}, 100*time.Millisecond).Result()
		assert.NotNil(t, err)
		assert.Nil(t, resp)
	})

	t.Run("when both sides simultaneously start writing", func(t *testing.T) {
		middleware.SetLogLevel("debug")
		time.Sleep(1 * time.Second)
		remotePing1 := actor.NewPID(types.NewRoutableAddress(host1.Identity(), host3.Identity()).String(), host3Ping.GetId())
		remotePing2 := actor.NewPID(types.NewRoutableAddress(host3.Identity(), host1.Identity()).String(), host1Ping.GetId())

		fut1 := rootContext.RequestFuture(remotePing1, &messages.Ping{Msg: "hi"}, 1000*time.Millisecond)
		fut2 := rootContext.RequestFuture(remotePing2, &messages.Ping{Msg: "hi"}, 1000*time.Millisecond)
		// time.Sleep(20 * time.Millisecond)
		// middleware.Log.Debugf("---------future 3--------------")
		// fut3 := rootContext.RequestFuture(remotePing1, &messages.Ping{Msg: "hi"}, 1000*time.Millisecond)

		// resp1, err := fut1.Result()
		// middleware.Log.Debugf("------------ tests over ------------------")
		// assert.NotNil(t, err)
		// assert.Nil(t, resp1)

		// _, err = fut2.Result()
		// require.NotNil(t, err)
		// // assert.Equal(t, resp2.(*messages.Pong).Msg, "hi")

		// resp3, err := fut3.Result()
		// require.Nil(t, err)
		// assert.Equal(t, resp3.(*messages.Pong).Msg, "hi")

		resp1, err := fut1.Result()
		middleware.Log.Debugf("------------ tests over ------------------")
		require.Nil(t, err)
		assert.Equal(t, resp1.(*messages.Pong).Msg, "hi")

		resp2, err := fut2.Result()
		require.Nil(t, err)
		assert.Equal(t, resp2.(*messages.Pong).Msg, "hi")
	})

}
