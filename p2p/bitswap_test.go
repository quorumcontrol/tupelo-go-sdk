package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/safewrap"

	"github.com/stretchr/testify/require"
)

func TestBitSwap(t *testing.T) {
	keyA, err := crypto.GenerateKey()
	require.Nil(t, err)
	keyB, err := crypto.GenerateKey()
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodeA, peerA, err := NewHostAndBitSwapPeer(ctx, WithKey(keyA))
	require.Nil(t, err)

	nodeB, peerB, err := NewHostAndBitSwapPeer(ctx, WithKey(keyB))

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

	log.Debugf("creating object: %s", n.Cid().String())

	swapCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.Debug("adding block")
	err = peerA.Add(swapCtx, n)
	require.Nil(t, err)

	log.Debug("looking for block")
	_, err = peerB.Get(swapCtx, n.Cid())
	if err != nil {
		t.Error(err)
	}
}
