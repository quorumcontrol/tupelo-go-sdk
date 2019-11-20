// +build integration

package client

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/quorumcontrol/messages/build/go/services"
	"github.com/quorumcontrol/messages/build/go/signatures"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/common/hexutil"
	cid "github.com/ipfs/go-cid"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/testhelpers"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type publicKeySet struct {
	BlsHexPublicKey   string `json:"blsHexPublicKey,omitempty"`
	EcdsaHexPublicKey string `json:"ecdsaHexPublicKey,omitempty"`
	PeerIDBase58Key   string `json:"peerIDBase58Key,omitempty"`
}

func loadSignerKeys() ([]*publicKeySet, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("No caller information")
	}
	jsonBytes, err := ioutil.ReadFile(path.Join(path.Dir(filename), "test-signer-keys/public-keys.json"))
	if err != nil {
		return nil, err
	}
	var keySet []*publicKeySet
	if err := json.Unmarshal(jsonBytes, &keySet); err != nil {
		return nil, err
	}

	return keySet, nil
}

func setupRemote(ctx context.Context, group *types.NotaryGroup) (p2p.Node, error) {
	remote.Start()
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("error generating key: %s", err)
	}
	p2pHost, err := p2p.NewLibP2PHost(ctx, key, 0)
	if err != nil {
		return nil, fmt.Errorf("error setting up p2p host: %s", err)
	}
	fmt.Println("bootstrapping with ", p2p.BootstrapNodes())
	if _, err = p2pHost.Bootstrap(p2p.BootstrapNodes()); err != nil {
		return nil, err
	}
	if err = p2pHost.WaitForBootstrap(1, 15*time.Second); err != nil {
		return nil, fmt.Errorf("just 1 bootstrap error: %v", err)
	}
	if err = p2pHost.WaitForBootstrap(2, 15*time.Second); err != nil {
		return nil, fmt.Errorf("just 1 bootstrap error: %v", err)
	}
	if err = p2pHost.WaitForBootstrap(len(group.Signers), 15*time.Second); err != nil {
		return nil, err
	}

	remote.NewRouter(p2pHost)
	group.SetupAllRemoteActors(&key.PublicKey)
	return p2pHost, nil
}

func setupNotaryGroup(ctx context.Context) (*types.NotaryGroup, error) {
	keys, err := loadSignerKeys()
	if err != nil {
		return nil, err
	}
	group := types.NewNotaryGroup("hardcodedprivatekeysareunsafe")
	for _, keySet := range keys {
		ecdsaBytes := hexutil.MustDecode(keySet.EcdsaHexPublicKey)
		verKeyBytes := hexutil.MustDecode(keySet.BlsHexPublicKey)
		ecdsaPubKey, err := crypto.UnmarshalPubkey(ecdsaBytes)
		if err != nil {
			return nil, err
		}
		signer := types.NewRemoteSigner(ecdsaPubKey, bls.BytesToVerKey(verKeyBytes))
		group.AddSigner(signer)
	}

	return group, nil
}

func TestBasicSetup(t *testing.T) {
	ctx := context.Background()

	remote.Start()
	defer remote.Stop()

	// Configuration setup
	bits, err := ioutil.ReadFile("../integration/configs/integration.toml")
	require.Nil(t, err)
	config, err := types.TomlToConfig(string(bits))
	require.Nil(t, err)
	notaryGroup, err := config.NotaryGroup(nil)
	require.Nil(t, err)

	// p2p setup
	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	p2pHost, err := p2p.NewLibP2PHost(ctx, key, 0)
	require.Nil(t, err)
	_, err = p2pHost.Bootstrap(config.BootstrapAddresses)
	require.Nil(t, err)
	err = p2pHost.WaitForBootstrap(1, 15*time.Second)
	require.Nil(t, err)
	remote.NewRouter(p2pHost)

	// Play a transaction on a chaintree
	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	chainTree, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
	require.Nil(t, err)

	client := New(notaryGroup, chainTree.MustId(), remote.NewNetworkPubSub(p2pHost.GetPubSub()))
	defer client.Stop()

	txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
	require.Nil(t, err)

	resp, err := client.PlayTransactions(chainTree, treeKey, nil, []*transactions.Transaction{txn})
	require.Nil(t, err)
	assert.Equal(t, resp.Tip.Bytes(), chainTree.Tip().Bytes())
}

func TestClientSendTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	host, err := setupRemote(ctx, ng)
	require.Nil(t, err)

	trans := testhelpers.NewValidTransaction(t)

	client := New(ng, string(trans.ObjectId), remote.NewNetworkPubSub(host.GetPubSub()))
	defer client.Stop()

	err = client.SendTransaction(&trans)
	require.Nil(t, err)
}

func TestClientSubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	host, err := setupRemote(ctx, ng)
	require.Nil(t, err)

	trans := testhelpers.NewValidTransaction(t)

	client := New(ng, string(trans.ObjectId), remote.NewNetworkPubSub(host.GetPubSub()))
	defer client.Stop()

	fut := client.Subscribe(&trans, 5*time.Second)

	time.Sleep(100 * time.Millisecond) // make sure the subscription completes

	err = client.SendTransaction(&trans)
	require.Nil(t, err)

	resp, err := fut.Result()
	require.Nil(t, err)
	require.NotNil(t, resp)
	require.IsType(t, &signatures.CurrentState{}, resp)
	currState := resp.(*signatures.CurrentState)
	assert.Equal(t, currState.Signature.NewTip, trans.NewTip)
}

func TestPlayTransactions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	host, err := setupRemote(ctx, ng)
	require.Nil(t, err)

	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	nodeStore := nodestore.MustMemoryStore(ctx)
	chain, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodeStore)
	require.Nil(t, err)

	client := New(ng, chain.MustId(), remote.NewNetworkPubSub(host.GetPubSub()))
	defer client.Stop()

	var remoteTip cid.Cid
	if !chain.IsGenesis() {
		remoteTip = chain.Tip()
	}

	txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
	require.Nil(t, err)

	resp, err := client.PlayTransactions(chain, treeKey, &remoteTip, []*transactions.Transaction{txn})
	require.Nil(t, err)
	assert.Equal(t, resp.Tip.Bytes(), chain.Tip().Bytes())

	t.Run("works on 2nd set", func(t *testing.T) {
		remoteTip := chain.Tip()
		txn2, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)
		resp, err := client.PlayTransactions(chain, treeKey, &remoteTip, []*transactions.Transaction{txn2})
		require.Nil(t, err)
		assert.Equal(t, resp.Tip.Bytes(), chain.Tip().Bytes())

		// and works a third time
		remoteTip = chain.Tip()
		txn3, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
		require.Nil(t, err)
		resp, err = client.PlayTransactions(chain, treeKey, &remoteTip, []*transactions.Transaction{txn3})
		require.Nil(t, err)
		assert.Equal(t, resp.Tip.Bytes(), chain.Tip().Bytes())
	})

	t.Run("works when setting a different branch of the tree", func(t *testing.T) {
		remoteTip := chain.Tip()
		txn2, err := chaintree.NewSetDataTransaction("over/yonder/in/the/hill", "sometestvalue")
		require.Nil(t, err)
		resp, err := client.PlayTransactions(chain, treeKey, &remoteTip, []*transactions.Transaction{txn2})
		require.Nil(t, err)
		assert.Equal(t, resp.Tip.Bytes(), chain.Tip().Bytes())
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

	blockWithHeaders, err := consensus.SignBlock(unsignedBlock, treeKey)
	require.Nil(t, err)

	_, err = tree.ChainTree.ProcessBlock(ctx, blockWithHeaders)
	require.Nil(t, err)

	return blockWithHeaders
}

func transactRemote(t testing.TB, client *Client, treeID string, blockWithHeaders *chaintree.BlockWithHeaders, newTip cid.Cid, stateNodes [][]byte, emptyTip cid.Cid) *actor.Future {
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

	fut := client.Subscribe(transMsg, 30*time.Second)
	time.Sleep(1 * time.Second)

	err := client.SendTransaction(transMsg)
	require.Nil(t, err)

	return fut
}

func TestSnoozedTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	host, err := setupRemote(ctx, ng)
	require.Nil(t, err)

	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	nodeStore := nodestore.MustMemoryStore(ctx)

	testTree, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodeStore)
	require.Nil(t, err)

	client1 := New(ng, testTree.MustId(), remote.NewNetworkPubSub(host.GetPubSub()))
	client1.Listen()
	defer client1.Stop()

	emptyTip := testTree.Tip()

	basisNodes0 := testhelpers.DagToByteNodes(t, testTree.ChainTree.Dag)

	blockWithHeaders0 := transactLocal(t, testTree, treeKey, 0, "down/in/the/tree", "atestvalue")
	tip0 := testTree.Tip()

	basisNodes1 := testhelpers.DagToByteNodes(t, testTree.ChainTree.Dag)

	blockWithHeaders1 := transactLocal(t, testTree, treeKey, 1, "other/thing", "sometestvalue")
	tip1 := testTree.Tip()

	sub1 := transactRemote(t, client1, testTree.MustId(), blockWithHeaders1, tip1, basisNodes1, emptyTip)

	time.Sleep(1 * time.Second)

	sub0 := transactRemote(t, client1, testTree.MustId(), blockWithHeaders0, tip0, basisNodes0, emptyTip)

	resp0, err := sub0.Result()
	require.Nil(t, err)
	require.IsType(t, &signatures.CurrentState{}, resp0)

	resp1, err := sub1.Result()
	require.Nil(t, err)
	require.IsType(t, &signatures.CurrentState{}, resp1)
}

func TestInvalidPreviousTipOnSnoozedTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	host, err := setupRemote(ctx, ng)
	require.Nil(t, err)

	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	nodeStoreA := nodestore.MustMemoryStore(ctx)
	nodeStoreB := nodestore.MustMemoryStore(ctx)

	testTreeA, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodeStoreA)
	require.Nil(t, err)

	clientA := New(ng, testTreeA.MustId(), remote.NewNetworkPubSub(host.GetPubSub()))
	clientA.Listen()
	defer clientA.Stop()

	// establish different first valid transactions on 2 different local chaintrees
	transactLocal(t, testTreeA, treeKey, 0, "down/in/the/treeA", "atestvalue")
	basisNodesA1 := testhelpers.DagToByteNodes(t, testTreeA.ChainTree.Dag)

	testTreeB, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodeStoreB)
	require.Nil(t, err)
	emptyTip := testTreeB.Tip()

	clientB := New(ng, testTreeB.MustId(), remote.NewNetworkPubSub(host.GetPubSub()))
	clientB.Listen()
	defer clientB.Stop()

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
	sub1 := transactRemote(t, clientB, testTreeB.MustId(), blockWithHeadersA1, tipA1, basisNodesA1, emptyTip)

	time.Sleep(1 * time.Second)

	sub0 := transactRemote(t, clientA, testTreeB.MustId(), blockWithHeadersB0, tipB0, basisNodesB0, emptyTip)

	resp0, err := sub0.Result()
	require.Nil(t, err)
	require.IsType(t, &signatures.CurrentState{}, resp0)

	t.Logf("resp0 tip %v", resp0.(*signatures.CurrentState).Signature.NewTip)

	_, err = sub1.Result()
	// TODO: this is now a timeout error.
	// we can probably figure out a more elegant way to test this - like maybe sending in a successful 3rd transaction
	require.NotNil(t, err)
}

func TestNonOwnerTransactions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	host, err := setupRemote(ctx, ng)
	require.Nil(t, err)

	treeKey1, err := crypto.GenerateKey()
	require.Nil(t, err)
	nodeStore := nodestore.MustMemoryStore(ctx)
	chain, err := consensus.NewSignedChainTree(treeKey1.PublicKey, nodeStore)
	require.Nil(t, err)

	client := New(ng, chain.MustId(), remote.NewNetworkPubSub(host.GetPubSub()))
	defer client.Stop()

	treeKey2, err := crypto.GenerateKey()
	require.Nil(t, err)

	// transaction with non-owner key should fail
	txn, err := chaintree.NewSetDataTransaction("down/in/the/thing", "sometestvalue")
	require.Nil(t, err)

	_, err = client.PlayTransactions(chain, treeKey2, nil, []*transactions.Transaction{txn})
	require.NotNil(t, err)
}
