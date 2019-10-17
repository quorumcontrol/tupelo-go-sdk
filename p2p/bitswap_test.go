package p2p

import (
	"context"
	"strconv"
	"testing"
	"time"

	logging "github.com/ipfs/go-log"

	"github.com/quorumcontrol/chaintree/safewrap"

	"github.com/stretchr/testify/require"
)

var testlog = logging.Logger("bitswaptest")

func TestBitSwap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodeA, peerA, err := NewHostAndBitSwapPeer(ctx)
	require.Nil(t, err)

	nodeB, peerB, err := NewHostAndBitSwapPeer(ctx)
	require.Nil(t, err)

	// Notice that the bootstrap is below the creation of the peer
	// THIS IS IMPORTANT

	_, err = nodeA.Bootstrap(bootstrapAddresses(nodeB))
	require.Nil(t, err)

	err = nodeA.WaitForBootstrap(1, 1*time.Second)
	require.Nil(t, err)

	_, err = nodeB.Bootstrap(bootstrapAddresses(nodeA))
	require.Nil(t, err)

	err = nodeB.WaitForBootstrap(1, 1*time.Second)
	require.Nil(t, err)

	sw := &safewrap.SafeWrap{}

	n := sw.WrapObject(map[string]string{"hello": "bitswap"})
	require.Nil(t, sw.Err)

	swapCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = peerA.Add(swapCtx, n)
	require.Nil(t, err)

	_, err = peerB.Get(swapCtx, n.Cid())
	if err != nil {
		t.Error(err)
	}
}

func TestBlockSwapping(t *testing.T) {
	logging.SetLogLevel("dht", "debug")
	logging.SetLogLevel("bitswaptest", "debug")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodeCount := 7
	nodes := make([]*LibP2PHost, nodeCount)
	peers := make([]*BitswapPeer, nodeCount)

	for i := 0; i < nodeCount; i++ {
		node, peer, err := NewHostAndBitSwapPeer(ctx)
		require.Nil(t, err)
		nodes[i] = node
		peers[i] = peer
	}

	for i := 1; i < nodeCount; i++ {
		_, err := nodes[i].Bootstrap(bootstrapAddresses(nodes[0]))
		require.Nil(t, err)
	}

	nodes[0].Bootstrap(bootstrapAddresses(nodes[1]))

	for i := 0; i < nodeCount; i++ {
		err := nodes[i].WaitForBootstrap(1, 2*time.Second)
		require.Nil(t, err)
	}

	sw := &safewrap.SafeWrap{}

	// n := sw.WrapObject(map[string]string{"hello": "bitswap"})
	// require.Nil(t, sw.Err)

	// swapCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	// defer cancel()

	testlog.Debug("running add")

	for i := 0; i < 75000; i++ {
		n := sw.WrapObject(map[string]string{"hello": "bitswap" + strconv.Itoa(i)})
		require.Nil(t, sw.Err)

		swapCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

		err := peers[0].Add(swapCtx, n)
		require.Nil(t, err)
		cancel()
	}
	testlog.Debug("---------------add finished")

	// _, err = peerB.Get(swapCtx, n.Cid())
	// if err != nil {
	// 	t.Error(err)
	// }
	select {}
}
