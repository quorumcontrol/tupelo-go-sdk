package client

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/v2/build/go/services"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/client/pubsubinterfaces/pubsubwrapper"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/testhelpers"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/quorumcontrol/tupelo-go-sdk/testnotarygroup"
	"github.com/quorumcontrol/tupelo/gossip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTupeloSystem(ctx context.Context, testSet *testnotarygroup.TestSet) (*types.NotaryGroup, []*gossip.Node, error) {
	nodes := make([]*gossip.Node, len(testSet.SignKeys))

	ng := types.NewNotaryGroup("testnotary")
	for i, signKey := range testSet.SignKeys {
		sk := signKey
		signer := types.NewRemoteSigner(testSet.PubKeys[i], sk.MustVerKey())
		ng.AddSigner(signer)
	}

	for i := range ng.AllSigners() {
		p2pNode, peer, err := p2p.NewHostAndBitSwapPeer(ctx, p2p.WithKey(testSet.EcdsaKeys[i]))
		if err != nil {
			return nil, nil, fmt.Errorf("error making node: %v", err)
		}

		n, err := gossip.NewNode(ctx, &gossip.NewNodeOptions{
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
	// setting log level to debug because it's useful output on test failures
	// this happens after the AllSigners loop because the node name is based on the
	// index in the signers
	for i := range ng.AllSigners() {
		if err := logging.SetLogLevel(fmt.Sprintf("node-%d", i), "debug"); err != nil {
			return nil, nil, fmt.Errorf("error setting log level: %v", err)
		}
	}

	return ng, nodes, nil
}

func startNodes(t *testing.T, ctx context.Context, nodes []*gossip.Node, bootAddrs []string) {
	for _, node := range nodes {
		err := node.Bootstrap(ctx, bootAddrs)
		require.Nil(t, err)
		err = node.Start(ctx)
		require.Nil(t, err)
	}
}

func TestClient(t *testing.T) {
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

		cli := New(group, pubsubwrapper.WrapLibp2p(cliHost.GetPubSub()), peer)
		// logging.SetLogLevel("g4-client", "debug")

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
		tree, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)

		proof, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn})
		require.Nil(t, err)
		assert.Equal(t, proof.Tip.Bytes(), tree.Tip().Bytes())
	})

	t.Run("test 3 subsequent transactions", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cli := newClient(ctx)

		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)
		tree, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)

		proof, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn})
		require.Nil(t, err)
		assert.Equal(t, proof.Tip.Bytes(), tree.Tip().Bytes())

		txn2, err := chaintree.NewSetDataTransaction("down/in/the/thing", "some other2")
		require.Nil(t, err)

		proof2, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn2})
		require.Nil(t, err)
		assert.Equal(t, proof2.Tip.Bytes(), tree.Tip().Bytes())

		txn3, err := chaintree.NewSetDataTransaction("down/in/the/thing", "some other3")
		require.Nil(t, err)

		proof3, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn3})
		require.Nil(t, err)
		assert.Equal(t, proof3.Tip.Bytes(), tree.Tip().Bytes())

	})

	t.Run("transactions played out of order succeed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cli := newClient(ctx)

		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)
		nodeStore := nodestore.MustMemoryStore(ctx)

		testTree, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodeStore)
		require.Nil(t, err)

		emptyTip := testTree.Tip()

		basisNodes0 := testhelpers.DagToByteNodes(t, testTree.ChainTree.Dag)

		blockWithHeaders0 := transactLocal(t, testTree, treeKey, 0, "down/in/the/tree", "atestvalue")
		tip0 := testTree.Tip()

		basisNodes1 := testhelpers.DagToByteNodes(t, testTree.ChainTree.Dag)

		blockWithHeaders1 := transactLocal(t, testTree, treeKey, 1, "other/thing", "sometestvalue")
		tip1 := testTree.Tip()

		respCh1, sub1 := transactRemote(ctx, t, cli, testTree.MustId(), blockWithHeaders1, tip1, basisNodes1, emptyTip)
		defer cli.UnsubscribeFromAbr(sub1)
		defer close(respCh1)

		respCh0, sub0 := transactRemote(ctx, t, cli, testTree.MustId(), blockWithHeaders0, tip0, basisNodes0, emptyTip)
		defer cli.UnsubscribeFromAbr(sub0)
		defer close(respCh0)

		resp0 := <-respCh1
		require.IsType(t, &Proof{}, resp0)

		resp1 := <-respCh0
		require.IsType(t, &Proof{}, resp1)
	})

	t.Run("invalid previous tip fails", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		clientA := newClient(ctx)
		clientB := newClient(ctx)

		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)
		nodeStoreA := nodestore.MustMemoryStore(ctx)
		nodeStoreB := nodestore.MustMemoryStore(ctx)

		testTreeA, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodeStoreA)
		require.Nil(t, err)

		// establish different first valid transactions on 2 different local chaintrees
		transactLocal(t, testTreeA, treeKey, 0, "down/in/the/treeA", "atestvalue")
		basisNodesA1 := testhelpers.DagToByteNodes(t, testTreeA.ChainTree.Dag)

		testTreeB, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodeStoreB)
		require.Nil(t, err)
		emptyTip := testTreeB.Tip()

		basisNodesB0 := testhelpers.DagToByteNodes(t, testTreeB.ChainTree.Dag)
		blockWithHeadersB0 := transactLocal(t, testTreeB, treeKey, 0, "down/in/the/treeB", "btestvalue")
		tipB0 := testTreeB.Tip()

		// run a second transaction on the first local chaintree
		blockWithHeadersA1 := transactLocal(t, testTreeA, treeKey, 1, "other/thing", "sometestvalue")
		tipA1 := testTreeA.Tip()

		/* Now send tx at height 1 from chaintree A followed by
		   tx at height 0 from chaintree B
		   tx at height 1 should be a byzantine transaction because its previous tip value
		   from chaintree A won't line up with tx at height 0 from chaintree B.
		   This can't be checked until after tx 0 is committed and this test is for
		   verifying that that happens and result is an invalid tx
		*/
		respCh1, sub1 := transactRemote(ctx, t, clientB, testTreeB.MustId(), blockWithHeadersA1, tipA1, basisNodesA1, emptyTip)
		defer clientB.UnsubscribeFromAbr(sub1)
		defer close(respCh1)

		time.Sleep(1 * time.Second)

		respCh0, sub0 := transactRemote(ctx, t, clientA, testTreeB.MustId(), blockWithHeadersB0, tipB0, basisNodesB0, emptyTip)
		defer clientA.UnsubscribeFromAbr(sub0)
		defer close(respCh0)

		resp0 := <-respCh0
		require.IsType(t, &Proof{}, resp0)

		// TODO: this is now a timeout error.
		// we can probably figure out a more elegant way to test this - like maybe sending in a successful 3rd transaction
		ticker := time.NewTimer(2 * time.Second)
		defer ticker.Stop()

		select {
		case <-respCh1:
			t.Fatalf("received a proof when we shouldn't have")
		case <-ticker.C:
			// yay a pass!
		}
	})

	t.Run("non-owner transactions fail", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cli := newClient(ctx)
		treeKey1, err := crypto.GenerateKey()
		require.Nil(t, err)
		nodeStore := nodestore.MustMemoryStore(ctx)
		chain, err := consensus.NewSignedChainTree(ctx, treeKey1.PublicKey, nodeStore)
		require.Nil(t, err)

		treeKey2, err := crypto.GenerateKey()
		require.Nil(t, err)

		// transaction with non-owner key should fail
		txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)

		_, err = cli.PlayTransactions(ctx, chain, treeKey2, []*transactions.Transaction{txn})
		require.NotNil(t, err)
	})

}

func transactLocal(t testing.TB, tree *consensus.SignedChainTree, treeKey *ecdsa.PrivateKey, height uint64, path, value string) *chaintree.BlockWithHeaders {
	ctx := context.TODO()
	var pt *cid.Cid
	if !tree.IsGenesis() {
		tip := tree.Tip()
		pt = &tip
	}

	txn, err := chaintree.NewSetDataTransaction(path, value)
	require.Nil(t, err)
	unsignedBlock := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  pt,
			Height:       height,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	blockWithHeaders, err := consensus.SignBlock(ctx, unsignedBlock, treeKey)
	require.Nil(t, err)

	_, err = tree.ChainTree.ProcessBlock(ctx, blockWithHeaders)
	require.Nil(t, err)

	return blockWithHeaders
}

func transactRemote(ctx context.Context, t testing.TB, client *Client, treeID string, blockWithHeaders *chaintree.BlockWithHeaders, newTip cid.Cid, stateNodes [][]byte, emptyTip cid.Cid) (chan *Proof, subscription) {
	sw := safewrap.SafeWrap{}

	var previousTipBytes []byte
	if blockWithHeaders.PreviousTip == nil {
		previousTipBytes = emptyTip.Bytes()
	} else {
		previousTipBytes = blockWithHeaders.PreviousTip.Bytes()
	}

	transMsg := &services.AddBlockRequest{
		PreviousTip: previousTipBytes,
		Height:      blockWithHeaders.Height,
		NewTip:      newTip.Bytes(),
		Payload:     sw.WrapObject(blockWithHeaders).RawData(),
		State:       stateNodes,
		ObjectId:    []byte(treeID),
	}

	t.Logf("sending remote transaction id: %s height: %d", base64.StdEncoding.EncodeToString(consensus.RequestID(transMsg)), transMsg.Height)

	resp := make(chan *Proof, 1)
	sub := client.SubscribeToAbr(ctx, transMsg, resp)

	err := client.SendWithoutWait(ctx, transMsg)
	require.Nil(t, err)

	return resp, sub
}