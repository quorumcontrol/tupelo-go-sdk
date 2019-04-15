package remote

import (
	"context"
	"crypto/ecdsa"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/p2p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPubSub(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	keys := make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < len(keys); i++ {
		ecdsaKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("error generating key: %v", err)
		}
		keys[i] = ecdsaKey
	}

	bootstrapper, err := p2p.NewLibP2PHost(ctx, keys[0], 0)
	require.Nil(t, err)

	nodeA, err := p2p.NewLibP2PHost(ctx, keys[1], 0)
	require.Nil(t, err)

	nodeB, err := p2p.NewLibP2PHost(ctx, keys[2], 0)
	require.Nil(t, err)

	_, err = nodeA.Bootstrap(bootstrapAddresses(bootstrapper))
	require.Nil(t, err)

	_, err = nodeB.Bootstrap(bootstrapAddresses(bootstrapper))
	require.Nil(t, err)

	err = nodeA.WaitForBootstrap(2, 1*time.Second)
	require.Nil(t, err)

	err = nodeB.WaitForBootstrap(2, 1*time.Second)
	require.Nil(t, err)

	actorContext := actor.EmptyRootContext

	tx := &messages.Transaction{
		ObjectID: []byte("totaltest"),
	}

	broadcaster := NewNetworkBroadcaster(nodeA)

	subscriber := actor.NewFuture(5 * time.Second)
	ready := actor.NewFuture(1 * time.Second)
	parent := func(actCtx actor.Context) {
		switch msg := actCtx.Message().(type) {
		case *actor.Started:
			actCtx.Spawn(NewNetworkSubscriberProps("testpubsub", nodeB))
			actCtx.Send(ready.PID(), true)
		case *messages.Transaction:
			actCtx.Send(subscriber.PID(), msg)
		}
	}

	receiver, err := actorContext.SpawnNamed(actor.PropsFromFunc(parent), "pubsubtest-receiver")
	require.Nil(t, err)
	defer receiver.Poison()

	_, err = ready.Result()
	require.Nil(t, err)

	err = broadcaster.Broadcast("testpubsub", tx)
	require.Nil(t, err)

	resp, err := subscriber.Result()
	require.Nil(t, err)

	assert.Equal(t, tx.ObjectID, resp.(*messages.Transaction).ObjectID)
}

func TestSimulatedBroadcaster(t *testing.T) {
	actorContext := actor.EmptyRootContext

	tx := &messages.Transaction{
		ObjectID: []byte("totaltest"),
	}

	broadcaster := NewSimulatedBroadcaster()

	subscriber := actor.NewFuture(3 * time.Second)
	ready := actor.NewFuture(500 * time.Millisecond)
	parent := func(actCtx actor.Context) {
		switch msg := actCtx.Message().(type) {
		case *actor.Started:
			actCtx.Spawn(broadcaster.NewSubscriberProps("testsimulatortopic"))
			time.Sleep(50 * time.Millisecond)
			actCtx.Send(ready.PID(), true)
		case *messages.Transaction:
			actCtx.Send(subscriber.PID(), msg)
		default:
			middleware.Log.Debugw("parent received message type", "type", reflect.TypeOf(msg).String())
		}
	}

	receiver, err := actorContext.SpawnNamed(actor.PropsFromFunc(parent), "pubsubtest-simulator-receiver")
	require.Nil(t, err)
	defer receiver.Poison()

	_, err = ready.Result()
	require.Nil(t, err)

	err = broadcaster.Broadcast("testsimulatortopic", tx)
	require.Nil(t, err)

	resp, err := subscriber.Result()
	require.Nil(t, err)

	assert.Equal(t, tx.ObjectID, resp.(*messages.Transaction).ObjectID)
}

func bootstrapAddresses(bootstrapHost p2p.Node) []string {
	addresses := bootstrapHost.Addresses()
	for _, addr := range addresses {
		addrStr := addr.String()
		if strings.Contains(addrStr, "127.0.0.1") {
			return []string{addrStr}
		}
	}
	return nil
}
