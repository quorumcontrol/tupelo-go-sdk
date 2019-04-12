package remote

import (
	"context"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/p2p"
	"github.com/quorumcontrol/tupelo/testnotarygroup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPubSub(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := testnotarygroup.NewTestSet(t, 3)

	bootstrapper, err := p2p.NewLibP2PHost(ctx, ts.EcdsaKeys[0], 0)
	require.Nil(t, err)

	nodeA, err := p2p.NewLibP2PHost(ctx, ts.EcdsaKeys[1], 0)
	require.Nil(t, err)

	nodeB, err := p2p.NewLibP2PHost(ctx, ts.EcdsaKeys[2], 0)
	require.Nil(t, err)

	_, err = nodeA.Bootstrap(testnotarygroup.BootstrapAddresses(bootstrapper))
	require.Nil(t, err)

	_, err = nodeB.Bootstrap(testnotarygroup.BootstrapAddresses(bootstrapper))
	require.Nil(t, err)

	err = nodeA.WaitForBootstrap(2, 1*time.Second)
	require.Nil(t, err)

	err = nodeB.WaitForBootstrap(2, 1*time.Second)
	require.Nil(t, err)

	subscriber := actor.NewFuture(5 * time.Second)

	actorContext := actor.EmptyRootContext

	middleware.SetLogLevel("debug")

	tx := &messages.Transaction{
		ObjectID: []byte("totaltest"),
	}

	broadcaster := NewNetworkBroadcaster(nodeA)

	receiver, err := actorContext.SpawnNamed(NewNetworkSubscriberProps(tx.TypeCode(), subscriber.PID(), nodeB), "pubsubtest-receiver")
	require.Nil(t, err)
	defer receiver.Poison()

	broadcaster.Broadcast(tx)

	resp, err := subscriber.Result()
	require.Nil(t, err)

	assert.Equal(t, tx.ObjectID, resp.(*messages.Transaction).ObjectID)
}
