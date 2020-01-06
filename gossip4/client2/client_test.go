package client2

import (
	"context"
	"fmt"
	"testing"

	g3types "github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/quorumcontrol/tupelo-go-sdk/testnotarygroup"
	"github.com/quorumcontrol/tupelo/gossip4"
	"github.com/stretchr/testify/require"
)

func newTupeloSystem(ctx context.Context, testSet *testnotarygroup.TestSet) (*g3types.NotaryGroup, []*gossip4.Node, error) {
	nodes := make([]*gossip4.Node, len(testSet.SignKeys))

	ng := g3types.NewNotaryGroup("testnotary")
	for i, signKey := range testSet.SignKeys {
		sk := signKey
		signer := g3types.NewRemoteSigner(testSet.PubKeys[i], sk.MustVerKey())
		ng.AddSigner(signer)
	}

	for i := range ng.AllSigners() {
		p2pNode, peer, err := p2p.NewHostAndBitSwapPeer(ctx, p2p.WithKey(testSet.EcdsaKeys[i])) // TODO: options?
		if err != nil {
			return nil, nil, fmt.Errorf("error making node: %v", err)
		}

		n, err := gossip4.NewNode(ctx, &gossip4.NewNodeOptions{
			P2PNode:     p2pNode,
			SignKey:     testSet.SignKeys[i],
			NotaryGroup: ng,
			DagStore:    peer,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("error making node: %v", err)
		}
		nodes[i] = n
	}

	return ng, nodes, nil
}

func startNodes(t *testing.T, ctx context.Context, nodes []*gossip4.Node) {
	booter, err := p2p.NewHostFromOptions(ctx)
	require.Nil(t, err)

	// logging.SetLogLevel(fmt.Sprintf("node-%d", n.signerIndex), "info")
	bootAddrs := make([]string, len(booter.Addresses()))
	for i, addr := range booter.Addresses() {
		bootAddrs[i] = addr.String()
	}

	for i, node := range nodes {
		if i > 0 {
			err := node.Bootstrap(ctx, bootAddrs)
			require.Nil(t, err)
			defer node.Close()
		}
		err := node.Start(ctx)
		require.Nil(t, err)
	}
}

func TestClient2(t *testing.T) {
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
}
