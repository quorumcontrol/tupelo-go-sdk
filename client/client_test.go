// +build integration

package client

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	cid "github.com/ipfs/go-cid"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/storage"
	"github.com/quorumcontrol/tupelo-go-client/bls"
	"github.com/quorumcontrol/tupelo-go-client/consensus"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/testhelpers"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-client/p2p"
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

func setupNotaryGroup(ctx context.Context) (*types.NotaryGroup, error) {
	remote.Start()
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("error generating key: %s", err)
	}
	p2pHost, err := p2p.NewLibP2PHost(ctx, key, 0)
	if err != nil {
		return nil, fmt.Errorf("error setting up p2p host: %s", err)
	}
	p2pHost.Bootstrap(p2p.BootstrapNodes())
	if err = p2pHost.WaitForBootstrap(1, 15*time.Second); err != nil {
		return nil, err
	}

	remote.NewRouter(p2pHost)

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
	group.SetupAllRemoteActors(&key.PublicKey)

	return group, nil
}

func TestClientSendTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	client := New(ng)
	defer client.Stop()

	trans := testhelpers.NewValidTransaction(t)
	err = client.SendTransaction(ng.GetRandomSigner(), &trans)
	require.Nil(t, err)
}

func TestClientSubscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	client := New(ng)
	defer client.Stop()

	trans := testhelpers.NewValidTransaction(t)

	newTip, err := cid.Cast(trans.NewTip)
	require.Nil(t, err)
	signer := ng.GetRandomSigner()
	ch, err := client.Subscribe(signer, string(trans.ObjectID), newTip, 5*time.Second)
	require.Nil(t, err)

	time.Sleep(100 * time.Millisecond) // make sure the subscription completes

	err = client.SendTransaction(signer, &trans)
	require.Nil(t, err)

	resp := <-ch
	require.NotNil(t, resp)
	require.IsType(t, &messages.CurrentState{}, resp)
	currState := resp.(*messages.CurrentState)
	assert.Equal(t, currState.Signature.NewTip, trans.NewTip)
}

func TestPlayTransactions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	client := New(ng)
	defer client.Stop()

	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	chain, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodeStore)

	var remoteTip cid.Cid
	if !chain.IsGenesis() {
		remoteTip = chain.Tip()
	}

	resp, err := client.PlayTransactions(chain, treeKey, &remoteTip, []*chaintree.Transaction{
		{
			Type: "SET_DATA",
			Payload: map[string]string{
				"path":  "down/in/the/thing",
				"value": "sometestvalue",
			},
		},
	})
	require.Nil(t, err)
	assert.Equal(t, resp.Tip.Bytes(), chain.Tip().Bytes())

	t.Run("works on 2nd set", func(t *testing.T) {
		remoteTip := chain.Tip()
		resp, err := client.PlayTransactions(chain, treeKey, &remoteTip, []*chaintree.Transaction{
			{
				Type: "SET_DATA",
				Payload: map[string]string{
					"path":  "down/in/the/thing",
					"value": "sometestvalue",
				},
			},
		})
		require.Nil(t, err)
		assert.Equal(t, resp.Tip.Bytes(), chain.Tip().Bytes())

		// and works a third time
		remoteTip = chain.Tip()
		resp, err = client.PlayTransactions(chain, treeKey, &remoteTip, []*chaintree.Transaction{
			{
				Type: "SET_DATA",
				Payload: map[string]string{
					"path":  "down/in/the/thing",
					"value": "sometestvalue",
				},
			},
		})
		require.Nil(t, err)
		assert.Equal(t, resp.Tip.Bytes(), chain.Tip().Bytes())
	})
}

func TestNonNilPreviousTipOnFirstTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	client := New(ng)
	defer client.Stop()

	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	chain1, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodeStore)
	require.Nil(t, err)

	var remoteTip cid.Cid
	if !chain1.IsGenesis() {
		remoteTip = chain1.Tip()
	}

	/* -----------------------------------------------------------------------
	   first transaction with a non-nil previous tip should fail
	   ----------------------------------------------------------------------- */

	// first valid transaction to get an otherwise-valid tip
	_, _ = client.PlayTransactions(chain1, treeKey, &remoteTip, []*chaintree.Transaction{
		{
			Type: "SET_DATA",
			Payload: map[string]string{
				"path":  "down/in/the/thing",
				"value": "sometestvalue",
			},
		},
	})

	// new chaintree to invalidate previous tip
	treeKey, err = crypto.GenerateKey()
	require.Nil(t, err)
	chain2, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodeStore)
	require.Nil(t, err)

	remoteTip = chain1.Tip()
	transaction := &chaintree.Transaction{
		Type: "SET_DATA",
		Payload: map[string]string{
			"path":  "down/in/the/thing",
			"value": "sometestvalue",
		},
	}
	unsignedBlock := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			Height:       0,
			PreviousTip:  &remoteTip,
			Transactions: []*chaintree.Transaction{transaction},
		},
	}
	blockWithHeaders, err := consensus.SignBlock(unsignedBlock, treeKey)
	require.Nil(t, err)

	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(treeKey.PublicKey).String())
	emptyTree := consensus.NewEmptyTree(treeDID, nodeStore)
	emptyTip := emptyTree.Tip

	nodes := testhelpers.DagToByteNodes(t, emptyTree)

	testTree, _ := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	_, _ = testTree.ProcessBlock(blockWithHeaders)

	sw := safewrap.SafeWrap{}
	transactionMsg := &messages.Transaction{
		PreviousTip: emptyTip.Bytes(),
		Height:      0,
		NewTip:      testTree.Dag.Tip.Bytes(),
		Payload:     sw.WrapObject(blockWithHeaders).RawData(),
		State:       nodes,
		ObjectID:    []byte(chain2.MustId()),
	}

	signer := ng.GetRandomSigner()
	respChan, err := client.Subscribe(signer, chain2.MustId(), cid.Undef, 5*time.Second)
	require.Nil(t, err)

	err = client.SendTransaction(signer, transactionMsg)
	require.Nil(t, err)

	resp := <-respChan

	require.IsType(t, &messages.Error{}, resp)
}

func transactLocal(t testing.TB, tree *consensus.SignedChainTree, treeKey *ecdsa.PrivateKey, height uint64, path, value string) *chaintree.BlockWithHeaders {
	var pt *cid.Cid
	if !tree.IsGenesis() {
		tip := tree.Tip()
		pt = &tip
	}

	unsignedBlock := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: pt,
			Height:      height,
			Transactions: []*chaintree.Transaction{
				{
					Type: "SET_DATA",
					Payload: map[string]string{
						"path":  path,
						"value": value,
					},
				},
			},
		},
	}

	blockWithHeaders, err := consensus.SignBlock(unsignedBlock, treeKey)
	require.Nil(t, err)

	_, err = tree.ChainTree.ProcessBlock(blockWithHeaders)
	require.Nil(t, err)

	return blockWithHeaders
}

func transactRemote(t testing.TB, client *Client, signer *types.Signer, treeID string, blockWithHeaders *chaintree.BlockWithHeaders, newTip cid.Cid, stateNodes [][]byte, emptyTip cid.Cid) chan interface{} {
	sw := safewrap.SafeWrap{}

	var previousTipBytes []byte
	if blockWithHeaders.PreviousTip == nil {
		previousTipBytes = emptyTip.Bytes()
	} else {
		previousTipBytes = blockWithHeaders.PreviousTip.Bytes()
	}

	transMsg := &messages.Transaction{
		PreviousTip: previousTipBytes,
		Height:      blockWithHeaders.Height,
		NewTip:      newTip.Bytes(),
		Payload:     sw.WrapObject(blockWithHeaders).RawData(),
		State:       stateNodes,
		ObjectID:    []byte(treeID),
	}

	respChan, err := client.Subscribe(signer, treeID, newTip, 30*time.Second)
	require.Nil(t, err)

	err = client.SendTransaction(signer, transMsg)
	require.Nil(t, err)

	return respChan
}

func TestSnoozedTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	client := New(ng)
	defer client.Stop()

	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())

	testTree, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodeStore)
	require.Nil(t, err)

	emptyTip := testTree.Tip()

	basisNodes0 := testhelpers.DagToByteNodes(t, testTree.ChainTree.Dag)

	blockWithHeaders0 := transactLocal(t, testTree, treeKey, 0, "down/in/the/tree", "atestvalue")
	tip0 := testTree.Tip()

	basisNodes1 := testhelpers.DagToByteNodes(t, testTree.ChainTree.Dag)

	blockWithHeaders1 := transactLocal(t, testTree, treeKey, 1, "other/thing", "sometestvalue")
	tip1 := testTree.Tip()

	signer := ng.GetRandomSigner()
	sub1 := transactRemote(t, client, signer, testTree.MustId(), blockWithHeaders1, tip1, basisNodes1, emptyTip)

	time.Sleep(1 * time.Second)

	sub0 := transactRemote(t, client, signer, testTree.MustId(), blockWithHeaders0, tip0, basisNodes0, emptyTip)

	resp0 := <-sub0
	require.IsType(t, &messages.CurrentState{}, resp0)

	resp1 := <-sub1
	require.IsType(t, &messages.CurrentState{}, resp1)
}

func TestInvalidPreviousTipOnSnoozedTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	client := New(ng)
	defer client.Stop()

	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	nodeStoreA := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	nodeStoreB := nodestore.NewStorageBasedStore(storage.NewMemStorage())

	testTreeA, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodeStoreA)
	require.Nil(t, err)

	// establish different first valid transactions on 2 different local chaintrees
	transactLocal(t, testTreeA, treeKey, 0, "down/in/the/treeA", "atestvalue")
	basisNodesA1 := testhelpers.DagToByteNodes(t, testTreeA.ChainTree.Dag)

	testTreeB, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodeStoreB)
	require.Nil(t, err)
	emptyTip := testTreeB.Tip()

	basisNodesB0 := testhelpers.DagToByteNodes(t, testTreeB.ChainTree.Dag)
	blockWithHeadersB0 := transactLocal(t, testTreeB, treeKey, 0, "down/in/the/treeB", "btestvalue")
	tipB0 := testTreeB.Tip()

	// run a second transaction on the first local chaintree
	blockWithHeadersA1 := transactLocal(t, testTreeA, treeKey, 1, "other/thing", "sometestvalue")
	tipA1 := testTreeA.Tip()

	/* Now send tx 1 from chaintree A to a signer for chaintree B followed by
	   tx 0 from chaintree B to the same signer.
	   tx1 should be a byzantine transaction because its previous tip value
	   from chaintree A won't line up with tx 0's from chaintree B.
	   This can't be checked until after tx 0 is committed and this test is for
	   verifying that that happens and results in an error response.
	*/
	signer := ng.GetRandomSigner()
	sub1 := transactRemote(t, client, signer, testTreeB.MustId(), blockWithHeadersA1, tipA1, basisNodesA1, emptyTip)

	time.Sleep(1 * time.Second)

	sub0 := transactRemote(t, client, signer, testTreeB.MustId(), blockWithHeadersB0, tipB0, basisNodesB0, emptyTip)

	resp0 := <-sub0
	require.IsType(t, &messages.CurrentState{}, resp0)

	resp1 := <-sub1
	require.IsType(t, &messages.Error{}, resp1)
	require.Equal(t, consensus.ErrInvalidTip, resp1.(*messages.Error).Code)
}

func TestNonOwnerTransactions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	client := New(ng)
	defer client.Stop()

	treeKey1, err := crypto.GenerateKey()
	require.Nil(t, err)
	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	chain, err := consensus.NewSignedChainTree(treeKey1.PublicKey, nodeStore)

	treeKey2, err := crypto.GenerateKey()
	require.Nil(t, err)

	// transaction with non-owner key should fail
	_, err = client.PlayTransactions(chain, treeKey2, nil, []*chaintree.Transaction{
		{
			Type: "SET_DATA",
			Payload: map[string]string{
				"path":  "down/in/the/thing",
				"value": "sometestvalue",
			},
		},
	})
	require.NotNil(t, err)

	// 2nd transaction with non-owner key should fail
	remoteTip := chain.Tip()
	_, err = client.PlayTransactions(chain, treeKey2, &remoteTip, []*chaintree.Transaction{
		{
			Type: "SET_DATA",
			Payload: map[string]string{
				"path":  "down/in/the/other/thing",
				"value": "someothertestvalue",
			},
		},
	})
	require.NotNil(t, err)
}
