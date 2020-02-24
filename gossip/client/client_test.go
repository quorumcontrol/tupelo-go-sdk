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

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/v2/build/go/gossip"
	"github.com/quorumcontrol/messages/v2/build/go/services"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/client/pubsubinterfaces/pubsubwrapper"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/testhelpers"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/quorumcontrol/tupelo-go-sdk/testnotarygroup"
	tupelogossip "github.com/quorumcontrol/tupelo/gossip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const groupMembers = 3

func newTupeloSystem(ctx context.Context, testSet *testnotarygroup.TestSet) (*types.NotaryGroup, []*tupelogossip.Node, error) {
	nodes := make([]*tupelogossip.Node, len(testSet.SignKeys))

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

		n, err := tupelogossip.NewNode(ctx, &tupelogossip.NewNodeOptions{
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

func startNodes(t *testing.T, ctx context.Context, nodes []*tupelogossip.Node, bootAddrs []string) {
	for _, node := range nodes {
		err := node.Bootstrap(ctx, bootAddrs)
		require.Nil(t, err)
		err = node.Start(ctx)
		require.Nil(t, err)
	}
}

func newClient(ctx context.Context, group *types.NotaryGroup, bootAddrs []string) (*Client, error) {
	cliHost, peer, err := p2p.NewHostAndBitSwapPeer(ctx)
	if err != nil {
		return nil, err
	}

	_, err = cliHost.Bootstrap(bootAddrs)
	if err != nil {
		return nil, err
	}

	err = cliHost.WaitForBootstrap(len(group.AllSigners()), 5*time.Second)
	if err != nil {
		return nil, err
	}

	cli := New(group, pubsubwrapper.WrapLibp2p(cliHost.GetPubSub()), peer)

	err = logging.SetLogLevel("g4-client", "debug")
	if err != nil {
		return nil, err
	}

	err = cli.Start(ctx)
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func TestClientSendTransactions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := testnotarygroup.NewTestSet(t, groupMembers)
	group, nodes, err := newTupeloSystem(ctx, ts)
	require.Nil(t, err)
	require.Len(t, nodes, groupMembers)

	booter, err := p2p.NewHostFromOptions(ctx)
	require.Nil(t, err)

	bootAddrs := make([]string, len(booter.Addresses()))
	for i, addr := range booter.Addresses() {
		bootAddrs[i] = addr.String()
	}

	startNodes(t, ctx, nodes, bootAddrs)

	t.Run("test basic setup", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cli, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)

		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)
		tree, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)

		proof, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn})
		require.Nil(t, err)
		assert.Equal(t, proof.Tip, tree.Tip().Bytes())
	})

	t.Run("test 3 subsequent transactions", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cli, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)

		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)
		tree, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)

		proof, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn})
		require.Nil(t, err)
		assert.Equal(t, proof.Tip, tree.Tip().Bytes())

		txn2, err := chaintree.NewSetDataTransaction("down/in/the/thing", "some other2")
		require.Nil(t, err)

		proof2, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn2})
		require.Nil(t, err)
		assert.Equal(t, proof2.Tip, tree.Tip().Bytes())

		txn3, err := chaintree.NewSetDataTransaction("down/in/the/thing", "some other3")
		require.Nil(t, err)

		proof3, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn3})
		require.Nil(t, err)
		assert.Equal(t, proof3.Tip, tree.Tip().Bytes())

	})

	t.Run("transactions played out of order succeed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cli, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)

		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)
		nodeStore := nodestore.MustMemoryStore(ctx)

		testTree, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodeStore)
		require.Nil(t, err)

		emptyTip := testTree.Tip()

		basisNodes0 := testhelpers.DagToByteNodes(t, testTree.ChainTree.Dag)

		blockWithHeaders0 := transactLocal(ctx, t, testTree, treeKey, 0, "down/in/the/tree", "atestvalue")
		tip0 := testTree.Tip()

		basisNodes1 := testhelpers.DagToByteNodes(t, testTree.ChainTree.Dag)

		blockWithHeaders1 := transactLocal(ctx, t, testTree, treeKey, 1, "other/thing", "sometestvalue")
		tip1 := testTree.Tip()

		respCh1, sub1 := transactRemote(ctx, t, cli, testTree.MustId(), blockWithHeaders1, tip1, basisNodes1, emptyTip)
		defer cli.UnsubscribeFromAbr(sub1)
		defer close(respCh1)

		respCh0, sub0 := transactRemote(ctx, t, cli, testTree.MustId(), blockWithHeaders0, tip0, basisNodes0, emptyTip)
		defer cli.UnsubscribeFromAbr(sub0)
		defer close(respCh0)

		resp0 := <-respCh0
		require.IsType(t, &gossip.Proof{}, resp0)

		resp1 := <-respCh1
		require.IsType(t, &gossip.Proof{}, resp1)

	})

	t.Run("invalid previous tip fails", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		clientA, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)
		clientB, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)

		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)
		nodeStoreA := nodestore.MustMemoryStore(ctx)
		nodeStoreB := nodestore.MustMemoryStore(ctx)

		testTreeA, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodeStoreA)
		require.Nil(t, err)

		// establish different first valid transactions on 2 different local chaintrees
		transactLocal(ctx, t, testTreeA, treeKey, 0, "down/in/the/treeA", "atestvalue")
		basisNodesA1 := testhelpers.DagToByteNodes(t, testTreeA.ChainTree.Dag)

		testTreeB, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodeStoreB)
		require.Nil(t, err)
		emptyTip := testTreeB.Tip()

		basisNodesB0 := testhelpers.DagToByteNodes(t, testTreeB.ChainTree.Dag)
		blockWithHeadersB0 := transactLocal(ctx, t, testTreeB, treeKey, 0, "down/in/the/treeB", "btestvalue")
		tipB0 := testTreeB.Tip()

		// run a second transaction on the first local chaintree
		blockWithHeadersA1 := transactLocal(ctx, t, testTreeA, treeKey, 1, "other/thing", "sometestvalue")
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
		require.IsType(t, &gossip.Proof{}, resp0)

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

		cli, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)

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

func transactLocal(ctx context.Context, t testing.TB, tree *consensus.SignedChainTree, treeKey *ecdsa.PrivateKey, height uint64, path, value string) *chaintree.BlockWithHeaders {
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

func transactRemote(ctx context.Context, t testing.TB, client *Client, treeID string, blockWithHeaders *chaintree.BlockWithHeaders, newTip cid.Cid, stateNodes [][]byte, emptyTip cid.Cid) (chan *gossip.Proof, subscription) {
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

	resp := make(chan *gossip.Proof, 1)
	sub, err := client.SubscribeToAbr(ctx, transMsg, resp)
	require.Nil(t, err)

	err = client.SendWithoutWait(ctx, transMsg)
	require.Nil(t, err)

	return resp, sub
}

func TestClientGetTip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := testnotarygroup.NewTestSet(t, groupMembers)
	group, nodes, err := newTupeloSystem(ctx, ts)
	require.Nil(t, err)
	require.Len(t, nodes, groupMembers)

	booter, err := p2p.NewHostFromOptions(ctx)
	require.Nil(t, err)

	bootAddrs := make([]string, len(booter.Addresses()))
	for i, addr := range booter.Addresses() {
		bootAddrs[i] = addr.String()
	}

	startNodes(t, ctx, nodes, bootAddrs)

	t.Run("test get existing tip", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cli, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)

		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)
		tree, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)

		sendProof, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn})
		require.Nil(t, err)
		assert.Equal(t, sendProof.Tip, tree.Tip().Bytes())

		proof, err := cli.GetTip(ctx, tree.MustId())
		require.Nil(t, err)

		require.Equal(t, sendProof.Tip, proof.Tip)
	})

	t.Run("get non existant tip", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cli, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)
		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)
		tree, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)

		sendProof, err := cli.PlayTransactions(ctx, tree, treeKey, []*transactions.Transaction{txn})
		require.Nil(t, err)
		assert.Equal(t, sendProof.Tip, tree.Tip().Bytes())

		_, err = cli.GetTip(ctx, "did:tupelo:doesnotexist")
		require.Equal(t, ErrNotFound, err)
	})
}

func TestTransactionAlreadyProcessed(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := testnotarygroup.NewTestSet(t, groupMembers)

	group, nodes, err := newTupeloSystem(parentCtx, ts)
	require.Nil(t, err)
	require.Len(t, nodes, groupMembers)

	booter, err := p2p.NewHostFromOptions(parentCtx)
	require.Nil(t, err)

	bootAddrs := make([]string, len(booter.Addresses()))
	for i, addr := range booter.Addresses() {
		bootAddrs[i] = addr.String()
	}

	startNodes(t, parentCtx, nodes, bootAddrs)

	t.Run("missed round still succeeds", func(t *testing.T) {
		ctx, cancel := context.WithCancel(parentCtx)
		defer cancel()

		// this test is a bit confusing to read, but in practice it's relatively simple
		// 1. create a client and send in a transaction (so it's accepted in a round)
		// 2. create a *new* client and send in the same transaction
		// this means that client2 won't see the first, accepted round
		// and it will go down into the hamt to make sure the transaction was accepted
		// which it was, so the test should pass

		cli1, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)

		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)

		blankTree1, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		blankTree2, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)

		proof, err := cli1.PlayTransactions(ctx, blankTree1, treeKey, []*transactions.Transaction{txn})
		require.Nil(t, err)
		assert.Equal(t, proof.Tip, blankTree1.Tip().Bytes())

		// now with tree2 - it will be the same ABR id, so it will not show up in the rounds,
		// but the test should still pass because it gets looked up in the hamt.

		cli2, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)

		// we have to do a dummy transaction in test to make sure round2 actually completes
		// we send this in a go routine since PlayTransactions is blocking
		go func() {
			dummyKey, err := crypto.GenerateKey()
			require.Nil(t, err)
			dummyTree, err := consensus.NewSignedChainTree(ctx, dummyKey.PublicKey, nodestore.MustMemoryStore(ctx))
			require.Nil(t, err)
			_, err = cli2.PlayTransactions(ctx, dummyTree, dummyKey, []*transactions.Transaction{txn})
			require.Nil(t, err)
		}()

		proof2, err := cli2.PlayTransactions(ctx, blankTree2, treeKey, []*transactions.Transaction{txn})
		require.Nil(t, err)
		assert.Equal(t, proof2.Tip, blankTree2.Tip().Bytes())
	})

	t.Run("fails when different transaction already accepted", func(t *testing.T) {
		ctx, cancel := context.WithCancel(parentCtx)
		defer cancel()

		// 1. send in a transaction to get accepted at height 0
		// 2. try to send in a differing transaction at height 0
		// 3. get the proof back (and an error) from the original Tx

		cli1, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)

		treeKey, err := crypto.GenerateKey()
		require.Nil(t, err)

		blankTree1, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)

		proof, err := cli1.PlayTransactions(ctx, blankTree1, treeKey, []*transactions.Transaction{txn})
		require.Nil(t, err)
		assert.Equal(t, proof.Tip, blankTree1.Tip().Bytes())

		cli1.logger.Debugf("cli1 played transaction, got tip: %s ", hexutil.Encode(proof.Tip))

		// now with tree2 - it will be the same ABR id, so it will not show up in the rounds,
		// but the test should still pass because it gets looked up in the hamt.

		cli2, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)

		blankTree2, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		differingTxn, err := chaintree.NewSetDataTransaction("down", "some very different value")
		require.Nil(t, err)

		// we have to do a dummy transaction in test to make sure round2 actually completes
		// we send this in a go routine since PlayTransactions is blocking
		go func() {
			dummyKey, err := crypto.GenerateKey()
			require.Nil(t, err)
			dummyTree, err := consensus.NewSignedChainTree(ctx, dummyKey.PublicKey, nodestore.MustMemoryStore(ctx))
			require.Nil(t, err)
			cli2.logger.Debugf("submitting dummy transaction")
			_, err = cli2.PlayTransactions(ctx, dummyTree, dummyKey, []*transactions.Transaction{txn})
			require.Nil(t, err)
		}()

		cli1.logger.Debugf("cli2 playing transaction", hexutil.Encode(proof.Tip))

		proof2, err := cli2.PlayTransactions(ctx, blankTree2, treeKey, []*transactions.Transaction{differingTxn})
		cli1.logger.Debugf("cli2 played transaction, got tip: %s ", hexutil.Encode(proof2.Tip))

		require.NotNil(t, err)
		assert.Equal(t, proof2.Tip, blankTree1.Tip().Bytes())
	})

}

func TestTokenTransactions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := testnotarygroup.NewTestSet(t, groupMembers)
	group, nodes, err := newTupeloSystem(ctx, ts)
	require.Nil(t, err)
	require.Len(t, nodes, groupMembers)

	booter, err := p2p.NewHostFromOptions(ctx)
	require.Nil(t, err)

	bootAddrs := make([]string, len(booter.Addresses()))
	for i, addr := range booter.Addresses() {
		bootAddrs[i] = addr.String()
	}

	startNodes(t, ctx, nodes, bootAddrs)

	sendKey, err := crypto.GenerateKey()
	require.Nil(t, err)

	receiveKey, err := crypto.GenerateKey()
	require.Nil(t, err)

	t.Run("valid transaction", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		sendCli, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)

		sendTree, err := consensus.NewSignedChainTree(ctx, sendKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		receiveTree, err := consensus.NewSignedChainTree(ctx, receiveKey.PublicKey, nodestore.MustMemoryStore(ctx))
		require.Nil(t, err)

		tokenName := "test-token"
		tokenMax := uint64(50)
		mintAmount := uint64(25)
		sendTxId := "send-test-transaction"
		sendAmount := uint64(10)

		establishTxn, err := chaintree.NewEstablishTokenTransaction(tokenName, tokenMax)
		require.Nil(t, err)

		mintTxn, err := chaintree.NewMintTokenTransaction(tokenName, mintAmount)
		require.Nil(t, err)

		sendTxn, err := chaintree.NewSendTokenTransaction(sendTxId, tokenName, sendAmount, receiveTree.MustId())
		require.Nil(t, err)

		senderTxns := []*transactions.Transaction{establishTxn, mintTxn, sendTxn}

		sendProof, err := sendCli.PlayTransactions(ctx, sendTree, sendKey, senderTxns)
		require.Nil(t, err)
		assert.Equal(t, sendProof.Tip, sendTree.Tip().Bytes())

		fmt.Printf("round height: %d, checkpoint cid: %s, state_cid: %s\n", sendProof.Round.Height, sendProof.Round.CheckpointCid, sendProof.Round.StateCid)
		fmt.Printf("round confirmation: %v\n\n\n", sendProof.RoundConfirmation)

		fullTokenName := &consensus.TokenName{ChainTreeDID: sendTree.MustId(), LocalName: tokenName}
		tokenPayload, err := consensus.TokenPayloadForTransaction(sendTree.ChainTree, fullTokenName, sendTxId, sendProof)
		require.Nil(t, err)
		assert.Equal(t, tokenPayload.Tip, sendTree.Tip().String())

		receiveCli, err := newClient(ctx, group, bootAddrs)
		require.Nil(t, err)

		tipCid, err := cid.Decode(tokenPayload.Tip)
		require.Nil(t, err)

		receiveTxn, err := chaintree.NewReceiveTokenTransaction(sendTxId, tipCid.Bytes(), tokenPayload.Proof, tokenPayload.Leaves)
		require.Nil(t, err)

		receiverTxn := []*transactions.Transaction{receiveTxn}

		_, err = receiveCli.PlayTransactions(ctx, receiveTree, receiveKey, receiverTxn)
		require.Nil(t, err)
	})
}
