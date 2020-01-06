package client2

import (
	"context"
	"fmt"
	"testing"
	"time"

	logging "github.com/ipfs/go-log"
	g3types "github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip4/testhelpers"
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
		logging.SetLogLevel(fmt.Sprintf("node-%d", i), "debug")

		p2pNode, peer, err := p2p.NewHostAndBitSwapPeer(ctx, p2p.WithKey(testSet.EcdsaKeys[i]), p2p.WithDiscoveryNamespaces("g4")) // TODO: options?
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

func startNodes(t *testing.T, ctx context.Context, nodes []*gossip4.Node, bootAddrs []string) {
	for _, node := range nodes {
		err := node.Bootstrap(ctx, bootAddrs)
		require.Nil(t, err)
		err = node.Start(ctx)
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

	booter, err := p2p.NewHostFromOptions(ctx)
	require.Nil(t, err)

	// logging.SetLogLevel(fmt.Sprintf("node-%d", n.signerIndex), "info")
	bootAddrs := make([]string, len(booter.Addresses()))
	for i, addr := range booter.Addresses() {
		bootAddrs[i] = addr.String()
	}

	startNodes(t, ctx, nodes, bootAddrs)

	cliHost, peer, err := p2p.NewHostAndBitSwapPeer(ctx, p2p.WithDiscoveryNamespaces("g4"))
	require.Nil(t, err)
	_, err = cliHost.Bootstrap(bootAddrs)
	require.Nil(t, err)

	err = cliHost.WaitForBootstrap(numMembers, 5*time.Second)
	require.Nil(t, err)

	cli := New(group, cliHost.GetPubSub(), peer)

	err = cli.Start(ctx)
	require.Nil(t, err)

	time.Sleep(1 * time.Second)

	abr1 := testhelpers.NewValidTransaction(t)

	err = cli.Send(ctx, &abr1, 5*time.Second)
	require.Nil(t, err)

}
