package client2

import (
	"context"
	"testing"

	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/quorumcontrol/tupelo-go-sdk/testnotarygroup"
	"github.com/stretchr/testify/require"
)

func TestRoundSubscriber(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numMembers := 3
	ts := testnotarygroup.NewTestSet(t, numMembers)
	group, nodes, err := newTupeloSystem(ctx, ts)
	require.Nil(t, err)
	require.Len(t, nodes, numMembers)

	startNodes(t, ctx, nodes)

	cliHost, peer, err := p2p.NewHostAndBitSwapPeer(ctx)
	require.Nil(t, err)

	cli := New(group, cliHost.GetPubSub(), peer)

	err = cli.Start(ctx)
	require.Nil(t, err)

	// sub := cli.subscriber

}
