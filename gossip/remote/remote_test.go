package remote

import (
	"context"
	"github.com/quorumcontrol/messages/v2/build/go/services"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/quorumcontrol/tupelo-go-sdk/testnotarygroup"
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
		case *services.Ping:
			ctx.Respond(&services.Pong{Msg: msg.Msg})
		}
	}))

	resp, err := rootContext.RequestFuture(localPing, &services.Ping{Msg: "hi"}, 1*time.Second).Result()
	require.Nil(t, err)
	assert.Equal(t, resp.(*services.Pong).Msg, "hi")
}

func TestRemoteMessageSending(t *testing.T) {
	ts := testnotarygroup.NewTestSet(t, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bootstrap := newBootstrapHost(ctx, t)

	host1, err := p2p.NewLibP2PHost(ctx, ts.EcdsaKeys[0], 0)
	require.Nil(t, err)
	_, err = host1.Bootstrap(testnotarygroup.BootstrapAddresses(bootstrap))
	require.Nil(t, err)

	host2, err := p2p.NewLibP2PHost(ctx, ts.EcdsaKeys[1], 0)
	require.Nil(t, err)
	_, err = host2.Bootstrap(testnotarygroup.BootstrapAddresses(bootstrap))
	require.Nil(t, err)

	host3, err := p2p.NewLibP2PHost(ctx, ts.EcdsaKeys[2], 0)
	require.Nil(t, err)
	_, err = host3.Bootstrap(testnotarygroup.BootstrapAddresses(bootstrap))
	require.Nil(t, err)

	err = host1.WaitForBootstrap(1, 5*time.Second)
	require.Nil(t, err)
	err = host2.WaitForBootstrap(1, 5*time.Second)
	require.Nil(t, err)
	err = host3.WaitForBootstrap(1, 5*time.Second)
	require.Nil(t, err)

	t.Logf("host1: %s / host2: %s / host3: %s", host1.Identity(), host2.Identity(), host3.Identity())

	pingFunc := func(ctx actor.Context) {
		switch msg := ctx.Message().(type) {
		case *services.Ping:
			// t.Logf("ctx: %v, msg: %v, sender: %v", ctx, msg, ctx.Sender().Address+ctx.Sender().GetId())
			ctx.Respond(&services.Pong{Msg: msg.Msg})
		}
	}

	rootContext := actor.EmptyRootContext

	host1Ping, err := rootContext.SpawnNamed(actor.PropsFromFunc(pingFunc), "ping-host1")
	require.Nil(t, err)
	defer actor.EmptyRootContext.Poison(host1Ping)

	host2Ping, err := rootContext.SpawnNamed(actor.PropsFromFunc(pingFunc), "ping-host2")
	require.Nil(t, err)
	defer actor.EmptyRootContext.Poison(host2Ping)

	host3Ping, err := rootContext.SpawnNamed(actor.PropsFromFunc(pingFunc), "ping-host3")
	require.Nil(t, err)
	defer actor.EmptyRootContext.Poison(host3Ping)

	Start()
	defer Stop()

	NewRouter(host1)
	NewRouter(host2)
	NewRouter(host3)

	t.Run("ping", func(t *testing.T) {
		remotePing := actor.NewPID(types.NewRoutableAddress(host1.Identity(), host3.Identity()).String(), host3Ping.GetId())

		resp, err := rootContext.RequestFuture(remotePing, &services.Ping{Msg: "hi"}, 100*time.Millisecond).Result()

		assert.Nil(t, err)
		assert.Equal(t, resp.(*services.Pong).Msg, "hi")
	})

	t.Run("sending a traceable when tracing is on", func(t *testing.T) {
		// We *should* have serializable and traceable messages, but we don't.
		// will take some work in the messages library (probaby with gogo protobuf)
		// in order to bring those back. Without those, it's really hard to test this functionality.
		t.Skip("we no longer have any serializable, and traceable messages at the moment.")
	})

	t.Run("when the otherside is closed permanently", func(t *testing.T) {
		newCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		host4, err := p2p.NewLibP2PHost(newCtx, ts.EcdsaKeys[3], 0)
		require.Nil(t, err)
		_, err = host4.Bootstrap(testnotarygroup.BootstrapAddresses(bootstrap))
		require.Nil(t, err)
		err = host4.WaitForBootstrap(2, 5*time.Second)
		require.Nil(t, err)
		host4Ping, err := rootContext.SpawnNamed(actor.PropsFromFunc(pingFunc), "ping-host4")
		require.Nil(t, err)

		remote4Ping := actor.NewPID(types.NewRoutableAddress(host1.Identity(), host4.Identity()).String(), host4Ping.GetId())

		NewRouter(host4)

		resp, err := rootContext.RequestFuture(remote4Ping, &services.Ping{Msg: "hi"}, 100*time.Millisecond).Result()
		require.Nil(t, err)
		assert.Equal(t, resp.(*services.Pong).Msg, "hi")

		defer actor.EmptyRootContext.Stop(host4Ping)
		cancel()

		resp, err = rootContext.RequestFuture(remote4Ping, &services.Ping{Msg: "hi"}, 100*time.Millisecond).Result()
		assert.NotNil(t, err)
		assert.Nil(t, resp)
	})

	// This test detected a previous race condition where two parties are trying to write to each
	// other at the same time. Previously, this caused a race condition where both sides would fail
	// to contact each other. We switched over to the separate incoming/outgoing streams in order
	// to address that race.
	t.Run("when both sides simultaneously start writing", func(t *testing.T) {

		remotePing1 := actor.NewPID(types.NewRoutableAddress(host1.Identity(), host3.Identity()).String(), host3Ping.GetId())
		remotePing2 := actor.NewPID(types.NewRoutableAddress(host3.Identity(), host1.Identity()).String(), host1Ping.GetId())

		fut1 := rootContext.RequestFuture(remotePing1, &services.Ping{Msg: "hi"}, 200*time.Millisecond)
		fut2 := rootContext.RequestFuture(remotePing2, &services.Ping{Msg: "hi"}, 200*time.Millisecond)

		resp1, err := fut1.Result()
		require.Nil(t, err)
		assert.Equal(t, resp1.(*services.Pong).Msg, "hi")

		resp2, err := fut2.Result()
		require.Nil(t, err)
		assert.Equal(t, resp2.(*services.Pong).Msg, "hi")
	})

}
