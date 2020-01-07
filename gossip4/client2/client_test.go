package client2

import (
	"context"
	"fmt"
	logging "github.com/ipfs/go-log"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip4/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/quorumcontrol/tupelo-go-sdk/testnotarygroup"
	"github.com/quorumcontrol/tupelo/gossip4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTupeloSystem(ctx context.Context, testSet *testnotarygroup.TestSet) (*types.NotaryGroup, []*gossip4.Node, error) {
	nodes := make([]*gossip4.Node, len(testSet.SignKeys))

	ng := types.NewNotaryGroup("testnotary")
	for i, signKey := range testSet.SignKeys {
		sk := signKey
		signer := types.NewRemoteSigner(testSet.PubKeys[i], sk.MustVerKey())
		ng.AddSigner(signer)
	}

	for i := range ng.AllSigners() {
		logging.SetLogLevel(fmt.Sprintf("node-%d", i), "debug")

		p2pNode, peer, err := p2p.NewHostAndBitSwapPeer(ctx, p2p.WithKey(testSet.EcdsaKeys[i]))
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

	bootAddrs := make([]string, len(booter.Addresses()))
	for i, addr := range booter.Addresses() {
		bootAddrs[i] = addr.String()
	}

	startNodes(t, ctx, nodes, bootAddrs)

	newClient := func(ctx context.Context) *Client {
		cliHost, peer, err := p2p.NewHostAndBitSwapPeer(ctx)
		require.Nil(t, err)
		_, err = cliHost.Bootstrap(bootAddrs)
		require.Nil(t, err)

		err = cliHost.WaitForBootstrap(numMembers, 5*time.Second)
		require.Nil(t, err)

		cli := New(group, cliHost.GetPubSub(), peer)

		err = cli.Start(ctx)
		require.Nil(t, err)
		return cli
	}

	t.Run("test basic setup", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cli := newClient(ctx)

		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)
		tree, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)

		proof, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn})
		require.Nil(t, err)
		assert.Equal(t, proof.Tip.Bytes(), tree.Tip().Bytes())
	})

	t.Run("test 2 subsequent transactions", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cli := newClient(ctx)
		logging.SetLogLevel("g4-client", "debug")

		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)
		tree, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)

		proof, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn})
		require.Nil(t, err)
		assert.Equal(t, proof.Tip.Bytes(), tree.Tip().Bytes())

		txn2, err := chaintree.NewSetDataTransaction("down/in/the/thing", "some other")
		require.Nil(t, err)

		proof2, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn2})
		require.Nil(t, err)
		assert.Equal(t, proof2.Tip.Bytes(), tree.Tip().Bytes())

	})

	// abr1 := testhelpers.NewValidTransaction(t)

	// _, err = cli.Send(ctx, &abr1, 5*time.Second)
	// require.Nil(t, err)
}
