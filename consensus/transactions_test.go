package consensus

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstablishTokenTransactionWithMaximum(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: nil,
			Transactions: []*chaintree.Transaction{
				{
					Type: "ESTABLISH_TOKEN",
					Payload: map[string]interface{}{
						"name": "testtoken",
						"monetaryPolicy": map[string]interface{}{
							"maximum": 42,
						},
					},
				},
			},
		},
	}

	testTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	assert.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)

	maximum, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", "testtoken", "monetaryPolicy", "maximum"})
	assert.Nil(t, err)
	assert.Equal(t, maximum, uint64(42))

	mints, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", "testtoken", "mints"})
	assert.Nil(t, err)
	assert.Nil(t, mints)

	sends, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", "testtoken", "sends"})
	assert.Nil(t, err)
	assert.Nil(t, sends)

	receives, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", "testtoken", "receives"})
	assert.Nil(t, err)
	assert.Nil(t, receives)
}

func TestMintToken(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: nil,
			Transactions: []*chaintree.Transaction{
				{
					Type: "ESTABLISH_TOKEN",
					Payload: map[string]interface{}{
						"name": "testtoken",
						"monetaryPolicy": map[string]interface{}{
							"maximum": 42,
						},
					},
				},
			},
		},
	}

	testTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	assert.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)

	mintBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: &testTree.Dag.Tip,
			Height:      1,
			Transactions: []*chaintree.Transaction{
				{
					Type: "MINT_TOKEN",
					Payload: map[string]interface{}{
						"name":   "testtoken",
						"amount": 40,
					},
				},
			},
		},
	}
	_, err = testTree.ProcessBlock(mintBlockWithHeaders)
	assert.Nil(t, err)

	mints, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", "testtoken", "mints"})
	assert.Nil(t, err)
	assert.Len(t, mints.([]interface{}), 1)

	// a 2nd mint succeeds when it's within the bounds of the maximum

	mintBlockWithHeaders2 := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: &testTree.Dag.Tip,
			Height:      2,
			Transactions: []*chaintree.Transaction{
				{
					Type: "MINT_TOKEN",
					Payload: map[string]interface{}{
						"name":   "testtoken",
						"amount": 1,
					},
				},
			},
		},
	}
	_, err = testTree.ProcessBlock(mintBlockWithHeaders2)
	assert.Nil(t, err)

	mints, _, err = testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", "testtoken", "mints"})
	assert.Nil(t, err)
	assert.Len(t, mints.([]interface{}), 2)

	// a third mint fails if it exceeds the max

	mintBlockWithHeaders3 := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: &testTree.Dag.Tip,
			Height:      2,
			Transactions: []*chaintree.Transaction{
				{
					Type: "MINT_TOKEN",
					Payload: map[string]interface{}{
						"name":   "testtoken",
						"amount": 100,
					},
				},
			},
		},
	}
	_, err = testTree.ProcessBlock(mintBlockWithHeaders3)
	assert.NotNil(t, err)

	mints, _, err = testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", "testtoken", "mints"})
	assert.Nil(t, err)
	assert.Len(t, mints.([]interface{}), 2)
}

func TestEstablishTokenTransactionWithoutMonetaryPolicy(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: nil,
			Transactions: []*chaintree.Transaction{
				{
					Type: "ESTABLISH_TOKEN",
					Payload: map[string]interface{}{
						"name": "testtoken",
					},
				},
			},
		},
	}

	testTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	assert.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)

	maximum, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", "testtoken", "monetaryPolicy", "maximum"})
	assert.Nil(t, err)
	assert.Empty(t, maximum)

	mints, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", "testtoken", "mints"})
	assert.Nil(t, err)
	assert.Nil(t, mints)

	sends, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", "testtoken", "sends"})
	assert.Nil(t, err)
	assert.Nil(t, sends)

	receives, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", "testtoken", "receives"})
	assert.Nil(t, err)
	assert.Nil(t, receives)
}

func TestSetData(t *testing.T) {
	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	treeDID := AddrToDid(crypto.PubkeyToAddress(treeKey.PublicKey).String())
	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	emptyTree := NewEmptyTree(treeDID, nodeStore)
	path := "some/data"
	value := "is now set"

	unsignedBlock := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: nil,
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
	testTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	require.Nil(t, err)

	blockWithHeaders, err := SignBlock(unsignedBlock, treeKey)
	require.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	require.Nil(t, err)

	// nodes, err := testTree.Dag.Nodes()
	// require.Nil(t, err)

	// assert the node the tree links to isn't itself a CID
	root, err := testTree.Dag.Get(testTree.Dag.Tip)
	assert.Nil(t, err)
	rootMap := make(map[string]interface{})
	err = cbornode.DecodeInto(root.RawData(), &rootMap)
	require.Nil(t, err)
	linkedTree, err := testTree.Dag.Get(rootMap["tree"].(cid.Cid))
	require.Nil(t, err)

	// assert that the thing being linked to isn't a CID itself
	treeCid := cid.Cid{}
	err = cbornode.DecodeInto(linkedTree.RawData(), &treeCid)
	assert.NotNil(t, err)

	// assert the thing being linked to is a map with data key
	treeMap := make(map[string]interface{})
	err = cbornode.DecodeInto(linkedTree.RawData(), &treeMap)
	assert.Nil(t, err)
	dataCid, ok := treeMap["data"]
	assert.True(t, ok)

	// assert the thing being linked to is a map with actual set data
	dataTree, err := testTree.Dag.Get(dataCid.(cid.Cid))
	assert.Nil(t, err)
	dataMap := make(map[string]interface{})
	err = cbornode.DecodeInto(dataTree.RawData(), &dataMap)
	assert.Nil(t, err)
	_, ok = dataMap["some"]
	assert.True(t, ok)

	// make sure the original data is still there after setting new data
	path = "other/data"
	value = "is also set"

	unsignedBlock = &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			Height:      1,
			PreviousTip: &testTree.Dag.Tip,
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

	blockWithHeaders, err = SignBlock(unsignedBlock, treeKey)
	require.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	require.Nil(t, err)

	dp, err := DecodePath("/tree/data/" + path)
	require.Nil(t, err)
	resp, remain, err := testTree.Dag.Resolve(dp)
	require.Nil(t, err)
	require.Len(t, remain, 0)
	require.Equal(t, value, resp)

	dp, err = DecodePath("/tree/data/some/data")
	require.Nil(t, err)
	resp, remain, err = testTree.Dag.Resolve(dp)
	require.Nil(t, err)
	require.Len(t, remain, 0)
	require.Equal(t, "is now set", resp)
}

func TestSetOwnership(t *testing.T) {
	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	keyAddr := crypto.PubkeyToAddress(treeKey.PublicKey).String()
	treeDID := AddrToDid(keyAddr)
	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	emptyTree := NewEmptyTree(treeDID, nodeStore)
	path := "some/data"
	value := "is now set"

	unsignedBlock := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: nil,
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
	testTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	require.Nil(t, err)

	blockWithHeaders, err := SignBlock(unsignedBlock, treeKey)
	require.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	require.Nil(t, err)

	dp, err := DecodePath("/tree/data/" + path)
	require.Nil(t, err)
	resp, remain, err := testTree.Dag.Resolve(dp)
	require.Nil(t, err)
	require.Len(t, remain, 0)
	require.Equal(t, value, resp)

	unsignedBlock = &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: &testTree.Dag.Tip,
			Height:      1,
			Transactions: []*chaintree.Transaction{
				{
					Type: "SET_OWNERSHIP",
					Payload: &SetOwnershipPayload{
						Authentication: []string{keyAddr},
					},
				},
			},
		},
	}
	blockWithHeaders, err = SignBlock(unsignedBlock, treeKey)
	require.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	require.Nil(t, err)

	resp, remain, err = testTree.Dag.Resolve(dp)
	require.Nil(t, err)
	assert.Len(t, remain, 0)
	assert.Equal(t, value, resp)
}

func TestSendToken(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	maximumAmount := uint64(50)
	height := uint64(0)
	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: nil,
			Height: height,
			Transactions: []*chaintree.Transaction{
				{
					Type: "ESTABLISH_TOKEN",
					Payload: map[string]interface{}{
						"name": "testtoken",
					},
				},
				{
					Type: "MINT_TOKEN",
					Payload: map[string]interface{}{
						"name":   "testtoken",
						"amount": maximumAmount,
					},
				},
			},
		},
	}

	testTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	assert.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	height++

	targetKey, err := crypto.GenerateKey()
	targetTreeDID := AddrToDid(crypto.PubkeyToAddress(targetKey.PublicKey).String())

	sendBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: &testTree.Dag.Tip,
			Height: height,
			Transactions: []*chaintree.Transaction{
				{
					Type: "SEND_TOKEN",
					Payload: map[string]interface{}{
						"id":          "1234",
						"name":        "testtoken",
						"amount":      30,
						"destination": targetTreeDID,
					},
				},
			},
		},
	}

	_, err = testTree.ProcessBlock(sendBlockWithHeaders)
	assert.Nil(t, err)
	height++

	sends, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", "testtoken", "sends", "0"})
	assert.Nil(t, err)
	assert.NotNil(t, sends)

	sendsMap := sends.(map[string]interface{})
	assert.Equal(t, sendsMap["id"], "1234")
	assert.Equal(t, sendsMap["amount"], uint64(30))
	lastSendAmount := sendsMap["amount"].(uint64)
	assert.Equal(t, sendsMap["destination"], targetTreeDID)

	overSpendBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip: &testTree.Dag.Tip,
			Height: height,
			Transactions: []*chaintree.Transaction{
				{
					Type: "SEND_TOKEN",
					Payload: map[string]interface{}{
						"id":          "1234",
						"name":        "testtoken",
						"amount":      (maximumAmount - lastSendAmount) + 1,
						"destination": targetTreeDID,
					},
				},
			},
		},
	}

	_, err = testTree.ProcessBlock(overSpendBlockWithHeaders)
	assert.NotNil(t, err)
}

func TestDecodePath(t *testing.T) {
	dp1, err := DecodePath("/some/data")
	assert.Nil(t, err)
	assert.Equal(t, []string{"some", "data"}, dp1)

	dp2, err := DecodePath("some/data")
	assert.Nil(t, err)
	assert.Equal(t, []string{"some", "data"}, dp2)

	dp3, err := DecodePath("//some/data")
	assert.NotNil(t, err)
	assert.Nil(t, dp3)

	dp4, err := DecodePath("/some//data")
	assert.NotNil(t, err)
	assert.Nil(t, dp4)

	dp5, err := DecodePath("/some/../data")
	assert.Nil(t, err)
	assert.Equal(t, []string{"some", "..", "data"}, dp5)

	dp6, err := DecodePath("")
	assert.Nil(t, err)
	assert.Equal(t, []string{}, dp6)

	dp7, err := DecodePath("/")
	assert.Nil(t, err)
	assert.Equal(t, []string{}, dp7)

	dp8, err := DecodePath("/_tupelo")
	assert.Nil(t, err)
	assert.Equal(t, []string{"_tupelo"}, dp8)

	dp9, err := DecodePath("_tupelo")
	assert.Nil(t, err)
	assert.Equal(t, []string{"_tupelo"}, dp9)

	dp10, err := DecodePath("//_tupelo")
	assert.NotNil(t, err)
	assert.Nil(t, dp10)
}
