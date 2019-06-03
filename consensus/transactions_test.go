package consensus_test

import (
	"strings"
	"testing"

	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"

	"github.com/ethereum/go-ethereum/crypto"
	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/chaintree/typecaster"
)

func TestEstablishTokenTransactionWithMaximum(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := consensus.NewEmptyTree(treeDID, store)

	tokenName := "testtoken"
	tokenFullName := strings.Join([]string{treeDID, tokenName}, ":")

	txn, err := chaintree.NewEstablishTokenTransaction(tokenName, 42)
	assert.Nil(t, err)

	blockWithHeaders := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	testTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	assert.Nil(t, err)

	_, err = testTree.ProcessBlock(&blockWithHeaders)
	assert.Nil(t, err)

	maximum, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", tokenFullName, "monetaryPolicy", "maximum"})
	assert.Nil(t, err)
	assert.Equal(t, maximum, uint64(42))

	mints, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", tokenFullName, "mints"})
	assert.Nil(t, err)
	assert.Nil(t, mints)

	sends, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", tokenFullName, "sends"})
	assert.Nil(t, err)
	assert.Nil(t, sends)

	receives, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", tokenFullName, "receives"})
	assert.Nil(t, err)
	assert.Nil(t, receives)
}

func TestMintToken(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := consensus.NewEmptyTree(treeDID, store)

	tokenName := "testtoken"
	tokenFullName := strings.Join([]string{treeDID, tokenName}, ":")

	txn, err := chaintree.NewEstablishTokenTransaction(tokenName, 42)
	assert.Nil(t, err)

	blockWithHeaders := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	testTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	assert.Nil(t, err)

	_, err = testTree.ProcessBlock(&blockWithHeaders)
	assert.Nil(t, err)

	mintTxn, err := chaintree.NewMintTokenTransaction(tokenName, 40)
	assert.Nil(t, err)

	mintBlockWithHeaders := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &testTree.Dag.Tip,
			Height:       1,
			Transactions: []*transactions.Transaction{mintTxn},
		},
	}
	_, err = testTree.ProcessBlock(&mintBlockWithHeaders)
	assert.Nil(t, err)

	mints, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", tokenFullName, "mints"})
	assert.Nil(t, err)
	assert.Len(t, mints.([]interface{}), 1)

	// a 2nd mint succeeds when it's within the bounds of the maximum

	mintTxn2, err := chaintree.NewMintTokenTransaction(tokenName, 1)
	assert.Nil(t, err)

	mintBlockWithHeaders2 := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &testTree.Dag.Tip,
			Height:       2,
			Transactions: []*transactions.Transaction{mintTxn2},
		},
	}
	_, err = testTree.ProcessBlock(&mintBlockWithHeaders2)
	assert.Nil(t, err)

	mints, _, err = testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", tokenFullName, "mints"})
	assert.Nil(t, err)
	assert.Len(t, mints.([]interface{}), 2)

	// a third mint fails if it exceeds the max

	mintTxn3, err := chaintree.NewMintTokenTransaction(tokenName, 100)
	assert.Nil(t, err)

	mintBlockWithHeaders3 := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &testTree.Dag.Tip,
			Height:       0,
			Transactions: []*transactions.Transaction{mintTxn3},
		},
	}
	_, err = testTree.ProcessBlock(&mintBlockWithHeaders3)
	assert.NotNil(t, err)

	mints, _, err = testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", tokenFullName, "mints"})
	assert.Nil(t, err)
	assert.Len(t, mints.([]interface{}), 2)
}

func TestEstablishTokenTransactionWithoutMonetaryPolicy(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := consensus.NewEmptyTree(treeDID, store)

	tokenName := "testtoken"
	tokenFullName := strings.Join([]string{treeDID, tokenName}, ":")

	payload := &transactions.EstablishTokenPayload{Name: tokenName}

	txn := &transactions.Transaction{
		Type:                  transactions.Transaction_ESTABLISHTOKEN,
		EstablishTokenPayload: payload,
	}

	blockWithHeaders := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	testTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	assert.Nil(t, err)

	_, err = testTree.ProcessBlock(&blockWithHeaders)
	assert.Nil(t, err)

	maximum, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", tokenFullName, "monetaryPolicy", "maximum"})
	assert.Nil(t, err)
	assert.Empty(t, maximum)

	mints, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", tokenFullName, "mints"})
	assert.Nil(t, err)
	assert.Nil(t, mints)

	sends, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", tokenFullName, "sends"})
	assert.Nil(t, err)
	assert.Nil(t, sends)

	receives, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", tokenFullName, "receives"})
	assert.Nil(t, err)
	assert.Nil(t, receives)
}

func TestSetData(t *testing.T) {
	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(treeKey.PublicKey).String())
	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	emptyTree := consensus.NewEmptyTree(treeDID, nodeStore)
	path := "some/data"
	value := "is now set"

	txn, err := chaintree.NewSetDataTransaction(path, value)
	assert.Nil(t, err)

	unsignedBlock := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}
	testTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	require.Nil(t, err)

	blockWithHeaders, err := consensus.SignBlock(&unsignedBlock, treeKey)
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

	txn, err = chaintree.NewSetDataTransaction(path, value)
	assert.Nil(t, err)

	unsignedBlock = chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			Height:       1,
			PreviousTip:  &testTree.Dag.Tip,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	blockWithHeaders, err = consensus.SignBlock(&unsignedBlock, treeKey)
	require.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	require.Nil(t, err)

	dp, err := consensus.DecodePath("/tree/data/" + path)
	require.Nil(t, err)
	resp, remain, err := testTree.Dag.Resolve(dp)
	require.Nil(t, err)
	require.Len(t, remain, 0)
	require.Equal(t, value, resp)

	dp, err = consensus.DecodePath("/tree/data/some/data")
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
	keyAddrs := []string{keyAddr}
	treeDID := consensus.AddrToDid(keyAddr)
	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	emptyTree := consensus.NewEmptyTree(treeDID, nodeStore)
	path := "some/data"
	value := "is now set"

	txn, err := chaintree.NewSetDataTransaction(path, value)
	assert.Nil(t, err)

	unsignedBlock := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}
	testTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	require.Nil(t, err)

	blockWithHeaders, err := consensus.SignBlock(&unsignedBlock, treeKey)
	require.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	require.Nil(t, err)

	dp, err := consensus.DecodePath("/tree/data/" + path)
	require.Nil(t, err)
	resp, remain, err := testTree.Dag.Resolve(dp)
	require.Nil(t, err)
	require.Len(t, remain, 0)
	require.Equal(t, value, resp)

	txn, err = chaintree.NewSetOwnershipTransaction(keyAddrs)
	assert.Nil(t, err)

	unsignedBlock = chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &testTree.Dag.Tip,
			Height:       1,
			Transactions: []*transactions.Transaction{txn},
		},
	}
	blockWithHeaders, err = consensus.SignBlock(&unsignedBlock, treeKey)
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
	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := consensus.NewEmptyTree(treeDID, store)

	tokenName := "testtoken"
	tokenFullName := strings.Join([]string{treeDID, tokenName}, ":")

	maximumAmount := uint64(50)
	height := uint64(0)

	establishTxn, err := chaintree.NewEstablishTokenTransaction(tokenName, 0)
	assert.Nil(t, err)

	mintTxn, err := chaintree.NewMintTokenTransaction(tokenName, maximumAmount)
	assert.Nil(t, err)

	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       height,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	testTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	assert.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	height++

	targetKey, err := crypto.GenerateKey()
	assert.Nil(t, err)
	targetTreeDID := consensus.AddrToDid(crypto.PubkeyToAddress(targetKey.PublicKey).String())

	txn, err := chaintree.NewSendTokenTransaction("1234", tokenName, 30, targetTreeDID)
	assert.Nil(t, err)

	sendBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &testTree.Dag.Tip,
			Height:       height,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	_, err = testTree.ProcessBlock(sendBlockWithHeaders)
	assert.Nil(t, err)
	height++

	sends, _, err := testTree.Dag.Resolve([]string{"tree", "_tupelo", "tokens", tokenFullName, "sends", "0"})
	assert.Nil(t, err)
	assert.NotNil(t, sends)

	sendsMap := sends.(map[string]interface{})
	assert.Equal(t, sendsMap["id"], "1234")
	assert.Equal(t, sendsMap["amount"], uint64(30))
	lastSendAmount := sendsMap["amount"].(uint64)
	assert.Equal(t, sendsMap["destination"], targetTreeDID)

	overSpendTxn, err := chaintree.NewSendTokenTransaction("1234", tokenName, (maximumAmount-lastSendAmount)+1, targetTreeDID)
	assert.Nil(t, err)

	overSpendBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &testTree.Dag.Tip,
			Height:       height,
			Transactions: []*transactions.Transaction{overSpendTxn},
		},
	}

	_, err = testTree.ProcessBlock(overSpendBlockWithHeaders)
	assert.NotNil(t, err)
}

func TestReceiveToken(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := consensus.NewEmptyTree(treeDID, store)

	tokenName1 := "testtoken1"

	maximumAmount := uint64(50)
	senderHeight := uint64(0)
	establishTxn, err := chaintree.NewEstablishTokenTransaction(tokenName1, 0)
	assert.Nil(t, err)

	mintTxn, err := chaintree.NewMintTokenTransaction(tokenName1, maximumAmount)
	assert.Nil(t, err)

	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	senderChainTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	assert.Nil(t, err)

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	// establish & mint another token to ensure we aren't relying on there only being one

	tokenName2 := "testtoken2"
	tokenFullName2 := strings.Join([]string{treeDID, tokenName2}, ":")

	establishTxn2, err := chaintree.NewEstablishTokenTransaction(tokenName2, 0)
	assert.Nil(t, err)

	mintTxn2, err := chaintree.NewMintTokenTransaction(tokenName2, maximumAmount/2)
	assert.Nil(t, err)

	blockWithHeaders = &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn2, mintTxn2},
		},
	}

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	recipientKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	recipientTreeDID := consensus.AddrToDid(crypto.PubkeyToAddress(recipientKey.PublicKey).String())

	sendTxn, err := chaintree.NewSendTokenTransaction("1234", tokenName2, 20, recipientTreeDID)
	assert.Nil(t, err)

	sendBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{sendTxn},
		},
	}

	_, err = senderChainTree.ProcessBlock(sendBlockWithHeaders)
	assert.Nil(t, err)

	recipientHeight := uint64(0)

	signedBlock, err := consensus.SignBlock(sendBlockWithHeaders, key)
	require.Nil(t, err)

	headers := &consensus.StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr := crypto.PubkeyToAddress(key.PublicKey).String()
	signature, ok := headers.Signatures[sigAddr]
	require.True(t, ok)

	tokenPath := []string{"tree", "_tupelo", "tokens", tokenFullName2, consensus.TokenSendLabel, "0"}
	leafNodes, err := senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves := make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	receiveTxn, err := chaintree.NewReceiveTokenTransaction("1234", senderChainTree.Dag.Tip.Bytes(), signature, leaves)
	assert.Nil(t, err)

	receiveBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	recipientChainTree, err := consensus.NewSignedChainTree(recipientKey.PublicKey, store)
	require.Nil(t, err)
	recipientChainTree.ChainTree.BlockValidators = append(recipientChainTree.ChainTree.BlockValidators, types.IsTokenRecipient)

	valid, err := recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.Nil(t, err)
	assert.True(t, valid)

	senderTree, err := senderChainTree.Tree()
	require.Nil(t, err)
	senderLedger := consensus.NewTreeLedger(senderTree, tokenFullName2)
	senderBalance, err := senderLedger.Balance()
	require.Nil(t, err)

	recipientTree, err := recipientChainTree.ChainTree.Tree()
	require.Nil(t, err)
	recipientLedger := consensus.NewTreeLedger(recipientTree, tokenFullName2)
	recipientBalance, err := recipientLedger.Balance()
	require.Nil(t, err)

	assert.Equal(t, uint64(5), senderBalance)
	assert.Equal(t, uint64(20), recipientBalance)
}

func TestReceiveTokenInvalidTip(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := consensus.NewEmptyTree(treeDID, store)

	tokenName1 := "testtoken1"

	maximumAmount := uint64(50)
	senderHeight := uint64(0)
	establishTxn, err := chaintree.NewEstablishTokenTransaction(tokenName1, 0)
	assert.Nil(t, err)

	mintTxn, err := chaintree.NewMintTokenTransaction(tokenName1, maximumAmount)
	assert.Nil(t, err)

	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	senderChainTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	assert.Nil(t, err)

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	// establish & mint another token to ensure we aren't relying on there only being one

	tokenName2 := "testtoken2"
	tokenFullName2 := strings.Join([]string{treeDID, tokenName2}, ":")

	establishTxn2, err := chaintree.NewEstablishTokenTransaction(tokenName2, 0)
	assert.Nil(t, err)

	mintTxn2, err := chaintree.NewMintTokenTransaction(tokenName2, maximumAmount/2)
	assert.Nil(t, err)

	blockWithHeaders = &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn2, mintTxn2},
		},
	}

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	recipientKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	recipientTreeDID := consensus.AddrToDid(crypto.PubkeyToAddress(recipientKey.PublicKey).String())

	sendTxn, err := chaintree.NewSendTokenTransaction("1234", tokenName2, 20, recipientTreeDID)
	assert.Nil(t, err)

	sendBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{sendTxn},
		},
	}

	_, err = senderChainTree.ProcessBlock(sendBlockWithHeaders)
	assert.Nil(t, err)

	recipientHeight := uint64(0)

	signedBlock, err := consensus.SignBlock(sendBlockWithHeaders, key)
	require.Nil(t, err)

	headers := &consensus.StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr := crypto.PubkeyToAddress(key.PublicKey).String()
	signature, ok := headers.Signatures[sigAddr]
	require.True(t, ok)

	tokenPath := []string{"tree", "_tupelo", "tokens", tokenFullName2, consensus.TokenSendLabel, "0"}
	leafNodes, err := senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves := make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	otherChainTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	require.Nil(t, err)

	receiveTxn, err := chaintree.NewReceiveTokenTransaction("1234", otherChainTree.Dag.Tip.Bytes(), signature, leaves)
	assert.Nil(t, err)

	receiveBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	recipientChainTree, err := consensus.NewSignedChainTree(recipientKey.PublicKey, store)
	require.Nil(t, err)
	recipientChainTree.ChainTree.BlockValidators = append(recipientChainTree.ChainTree.BlockValidators, types.IsTokenRecipient)

	valid, err := recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.NotNil(t, err)
	assert.False(t, valid)
}

func TestReceiveTokenInvalidDoubleReceive(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := consensus.NewEmptyTree(treeDID, store)

	tokenName1 := "testtoken1"

	maximumAmount := uint64(50)
	senderHeight := uint64(0)
	establishTxn, err := chaintree.NewEstablishTokenTransaction(tokenName1, 0)
	assert.Nil(t, err)

	mintTxn, err := chaintree.NewMintTokenTransaction(tokenName1, maximumAmount)
	assert.Nil(t, err)

	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	senderChainTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	assert.Nil(t, err)

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	// establish & mint another token to ensure we aren't relying on there only being one

	tokenName2 := "testtoken2"
	tokenFullName2 := strings.Join([]string{treeDID, tokenName2}, ":")

	establishTxn2, err := chaintree.NewEstablishTokenTransaction(tokenName2, 0)
	assert.Nil(t, err)

	mintTxn2, err := chaintree.NewMintTokenTransaction(tokenName2, maximumAmount/2)
	assert.Nil(t, err)

	blockWithHeaders = &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn2, mintTxn2},
		},
	}

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	recipientKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	recipientTreeDID := consensus.AddrToDid(crypto.PubkeyToAddress(recipientKey.PublicKey).String())

	sendTxn, err := chaintree.NewSendTokenTransaction("1234", tokenName2, 20, recipientTreeDID)
	assert.Nil(t, err)

	sendBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{sendTxn},
		},
	}

	_, err = senderChainTree.ProcessBlock(sendBlockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	recipientHeight := uint64(0)

	signedBlock, err := consensus.SignBlock(sendBlockWithHeaders, key)
	require.Nil(t, err)

	headers := &consensus.StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr := crypto.PubkeyToAddress(key.PublicKey).String()
	signature, ok := headers.Signatures[sigAddr]
	require.True(t, ok)

	tokenPath := []string{"tree", "_tupelo", "tokens", tokenFullName2, consensus.TokenSendLabel, "0"}
	leafNodes, err := senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves := make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	receiveTxn, err := chaintree.NewReceiveTokenTransaction("1234", senderChainTree.Dag.Tip.Bytes(), signature, leaves)
	assert.Nil(t, err)

	receiveBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	recipientChainTree, err := consensus.NewSignedChainTree(recipientKey.PublicKey, store)
	require.Nil(t, err)
	recipientChainTree.ChainTree.BlockValidators = append(recipientChainTree.ChainTree.BlockValidators, types.IsTokenRecipient)

	valid, err := recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.Nil(t, err)
	assert.True(t, valid)
	recipientHeight++

	// now attempt to receive a new, otherwise valid send w/ the same transaction id (which should fail)

	sendTxn2, err := chaintree.NewSendTokenTransaction("1234", tokenName2, 2, recipientTreeDID)
	assert.Nil(t, err)

	sendBlockWithHeaders = &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{sendTxn2},
		},
	}

	_, err = senderChainTree.ProcessBlock(sendBlockWithHeaders)
	assert.Nil(t, err)

	signedBlock, err = consensus.SignBlock(sendBlockWithHeaders, key)
	require.Nil(t, err)

	headers = &consensus.StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr = crypto.PubkeyToAddress(key.PublicKey).String()
	signature, ok = headers.Signatures[sigAddr]
	require.True(t, ok)

	tokenPath = []string{"tree", "_tupelo", "tokens", tokenFullName2, consensus.TokenSendLabel, "1"}
	leafNodes, err = senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves = make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	receiveTxn, err = chaintree.NewReceiveTokenTransaction("1234", senderChainTree.Dag.Tip.Bytes(), signature, leaves)
	assert.Nil(t, err)

	receiveBlockWithHeaders = &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &recipientChainTree.ChainTree.Dag.Tip,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	valid, err = recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.NotNil(t, err)
	assert.False(t, valid)
}

func TestReceiveTokenInvalidSignature(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := consensus.NewEmptyTree(treeDID, store)

	tokenName1 := "testtoken1"

	maximumAmount := uint64(50)
	senderHeight := uint64(0)
	establishTxn, err := chaintree.NewEstablishTokenTransaction(tokenName1, 0)
	assert.Nil(t, err)

	mintTxn, err := chaintree.NewMintTokenTransaction(tokenName1, maximumAmount)
	assert.Nil(t, err)

	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	senderChainTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	assert.Nil(t, err)

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	// establish & mint another token to ensure we aren't relying on there only being one

	tokenName2 := "testtoken2"
	tokenFullName2 := strings.Join([]string{treeDID, tokenName2}, ":")

	establishTxn2, err := chaintree.NewEstablishTokenTransaction(tokenName2, 0)
	assert.Nil(t, err)

	mintTxn2, err := chaintree.NewMintTokenTransaction(tokenName2, maximumAmount/2)
	assert.Nil(t, err)

	blockWithHeaders = &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn2, mintTxn2},
		},
	}

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	recipientKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	recipientTreeDID := consensus.AddrToDid(crypto.PubkeyToAddress(recipientKey.PublicKey).String())

	sendTxn, err := chaintree.NewSendTokenTransaction("1234", tokenName2, 20, recipientTreeDID)
	assert.Nil(t, err)

	sendBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{sendTxn},
		},
	}

	_, err = senderChainTree.ProcessBlock(sendBlockWithHeaders)
	assert.Nil(t, err)

	otherKey, err := crypto.GenerateKey()
	require.Nil(t, err)

	signedBlock, err := consensus.SignBlock(sendBlockWithHeaders, otherKey)
	require.Nil(t, err)

	headers := &consensus.StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr := crypto.PubkeyToAddress(otherKey.PublicKey).String()
	signature, ok := headers.Signatures[sigAddr]
	require.True(t, ok)

	objectID, err := senderChainTree.Id()
	require.Nil(t, err)
	signature.ObjectId = []byte(objectID)
	signature.NewTip = senderChainTree.Dag.Tip.Bytes()
	signature.PreviousTip = signedBlock.PreviousTip.Bytes()

	tokenPath := []string{"tree", "_tupelo", "tokens", tokenFullName2, consensus.TokenSendLabel, "0"}
	leafNodes, err := senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves := make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	recipientHeight := uint64(0)

	receiveTxn, err := chaintree.NewReceiveTokenTransaction("1234", senderChainTree.Dag.Tip.Bytes(), signature, leaves)
	assert.Nil(t, err)

	receiveBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	recipientChainTree, err := consensus.NewSignedChainTree(recipientKey.PublicKey, store)
	require.Nil(t, err)

	isValidSignature := types.GenerateIsValidSignature(func(sig *signatures.Signature) (bool, error) {
		return false, nil
	})

	recipientChainTree.ChainTree.BlockValidators = append(recipientChainTree.ChainTree.BlockValidators,
		types.IsTokenRecipient, isValidSignature)

	valid, err := recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.Nil(t, err)
	assert.False(t, valid)
}

func TestReceiveTokenInvalidDestinationChainId(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := consensus.NewEmptyTree(treeDID, store)

	tokenName1 := "testtoken1"

	maximumAmount := uint64(50)
	senderHeight := uint64(0)

	establishTxn, err := chaintree.NewEstablishTokenTransaction(tokenName1, 0)
	assert.Nil(t, err)

	mintTxn, err := chaintree.NewMintTokenTransaction(tokenName1, maximumAmount)
	assert.Nil(t, err)

	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	senderChainTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	assert.Nil(t, err)

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	// establish & mint another token to ensure we aren't relying on there only being one

	tokenName2 := "testtoken2"
	tokenFullName2 := strings.Join([]string{treeDID, tokenName2}, ":")

	establishTxn2, err := chaintree.NewEstablishTokenTransaction(tokenName2, 0)
	assert.Nil(t, err)

	mintTxn2, err := chaintree.NewMintTokenTransaction(tokenName2, maximumAmount/2)
	assert.Nil(t, err)

	blockWithHeaders = &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn2, mintTxn2},
		},
	}

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	otherKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	otherTreeDID := consensus.AddrToDid(crypto.PubkeyToAddress(otherKey.PublicKey).String())

	sendTxn, err := chaintree.NewSendTokenTransaction("1234", tokenName2, 20, otherTreeDID)
	assert.Nil(t, err)

	sendBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{sendTxn},
		},
	}

	_, err = senderChainTree.ProcessBlock(sendBlockWithHeaders)
	assert.Nil(t, err)

	signedBlock, err := consensus.SignBlock(sendBlockWithHeaders, key)
	require.Nil(t, err)

	headers := &consensus.StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr := crypto.PubkeyToAddress(key.PublicKey).String()
	signature, ok := headers.Signatures[sigAddr]
	require.True(t, ok)

	tokenPath := []string{"tree", "_tupelo", "tokens", tokenFullName2, consensus.TokenSendLabel, "0"}
	leafNodes, err := senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves := make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	recipientHeight := uint64(0)

	receiveTxn, err := chaintree.NewReceiveTokenTransaction("1234", senderChainTree.Dag.Tip.Bytes(), signature, leaves)
	assert.Nil(t, err)

	receiveBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	recipientKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	recipientChainTree, err := consensus.NewSignedChainTree(recipientKey.PublicKey, store)
	require.Nil(t, err)
	recipientChainTree.ChainTree.BlockValidators = append(recipientChainTree.ChainTree.BlockValidators, types.IsTokenRecipient)

	valid, err := recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.Nil(t, err)
	assert.False(t, valid)
}

func TestReceiveTokenMismatchedSignatureTip(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := consensus.NewEmptyTree(treeDID, store)

	tokenName1 := "testtoken1"

	maximumAmount := uint64(50)
	senderHeight := uint64(0)
	establishTxn, err := chaintree.NewEstablishTokenTransaction(tokenName1, 0)
	assert.Nil(t, err)

	mintTxn, err := chaintree.NewMintTokenTransaction(tokenName1, maximumAmount)
	assert.Nil(t, err)

	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	senderChainTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	assert.Nil(t, err)

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	// establish & mint another token to ensure we aren't relying on there only being one

	tokenName2 := "testtoken2"
	tokenFullName2 := strings.Join([]string{treeDID, tokenName2}, ":")

	establishTxn2, err := chaintree.NewEstablishTokenTransaction(tokenName2, 0)
	assert.Nil(t, err)

	mintTxn2, err := chaintree.NewMintTokenTransaction(tokenName2, maximumAmount/2)
	assert.Nil(t, err)

	blockWithHeaders = &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn2, mintTxn2},
		},
	}

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	recipientKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	recipientTreeDID := consensus.AddrToDid(crypto.PubkeyToAddress(recipientKey.PublicKey).String())

	sendTxn, err := chaintree.NewSendTokenTransaction("1234", tokenName2, 20, recipientTreeDID)
	assert.Nil(t, err)

	sendBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{sendTxn},
		},
	}

	_, err = senderChainTree.ProcessBlock(sendBlockWithHeaders)
	assert.Nil(t, err)

	otherKey, err := crypto.GenerateKey()
	require.Nil(t, err)

	signedBlock, err := consensus.SignBlock(sendBlockWithHeaders, otherKey)
	require.Nil(t, err)

	headers := &consensus.StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr := crypto.PubkeyToAddress(otherKey.PublicKey).String()
	signature, ok := headers.Signatures[sigAddr]
	require.True(t, ok)

	objectID, err := senderChainTree.Id()
	require.Nil(t, err)
	signature.ObjectId = []byte(objectID)
	signature.NewTip = emptyTree.Tip.Bytes() // invalid
	signature.PreviousTip = signedBlock.PreviousTip.Bytes()

	tokenPath := []string{"tree", "_tupelo", "tokens", tokenFullName2, consensus.TokenSendLabel, "0"}
	leafNodes, err := senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves := make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	recipientHeight := uint64(0)

	receiveTxn, err := chaintree.NewReceiveTokenTransaction("1234", senderChainTree.Dag.Tip.Bytes(), signature, leaves)
	assert.Nil(t, err)

	receiveBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	recipientChainTree, err := consensus.NewSignedChainTree(recipientKey.PublicKey, store)
	require.Nil(t, err)

	isValidSignature := types.GenerateIsValidSignature(func(sig *signatures.Signature) (bool, error) {
		return true, nil // this should get caught before it gets here; so ensure this doesn't cause false positives
	})

	recipientChainTree.ChainTree.BlockValidators = append(recipientChainTree.ChainTree.BlockValidators,
		types.IsTokenRecipient, isValidSignature)

	valid, err := recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.Nil(t, err)
	assert.False(t, valid)
}

func TestDecodePath(t *testing.T) {
	dp1, err := consensus.DecodePath("/some/data")
	assert.Nil(t, err)
	assert.Equal(t, []string{"some", "data"}, dp1)

	dp2, err := consensus.DecodePath("some/data")
	assert.Nil(t, err)
	assert.Equal(t, []string{"some", "data"}, dp2)

	dp3, err := consensus.DecodePath("//some/data")
	assert.NotNil(t, err)
	assert.Nil(t, dp3)

	dp4, err := consensus.DecodePath("/some//data")
	assert.NotNil(t, err)
	assert.Nil(t, dp4)

	dp5, err := consensus.DecodePath("/some/../data")
	assert.Nil(t, err)
	assert.Equal(t, []string{"some", "..", "data"}, dp5)

	dp6, err := consensus.DecodePath("")
	assert.Nil(t, err)
	assert.Equal(t, []string{}, dp6)

	dp7, err := consensus.DecodePath("/")
	assert.Nil(t, err)
	assert.Equal(t, []string{}, dp7)

	dp8, err := consensus.DecodePath("/_tupelo")
	assert.Nil(t, err)
	assert.Equal(t, []string{"_tupelo"}, dp8)

	dp9, err := consensus.DecodePath("_tupelo")
	assert.Nil(t, err)
	assert.Equal(t, []string{"_tupelo"}, dp9)

	dp10, err := consensus.DecodePath("//_tupelo")
	assert.NotNil(t, err)
	assert.Nil(t, dp10)
}
