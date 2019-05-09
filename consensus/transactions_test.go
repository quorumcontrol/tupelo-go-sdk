package consensus

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/ptypes"
	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/transactions"
	"github.com/quorumcontrol/storage"
	"github.com/quorumcontrol/tupelo-go-sdk/conversion"
	"github.com/quorumcontrol/tupelo-go-sdk/testfakes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/chaintree/typecaster"

	extmsgs "github.com/quorumcontrol/tupelo-go-sdk/gossip3/messages"
)

func TestEstablishTokenTransactionWithMaximum(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	tokenName := "testtoken"
	tokenFullName := strings.Join([]string{treeDID, tokenName}, ":")

	txn := testfakes.EstablishTokenTransaction(tokenName, 42)
	blockWithHeaders := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	testTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
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
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	tokenName := "testtoken"
	tokenFullName := strings.Join([]string{treeDID, tokenName}, ":")

	txn := testfakes.EstablishTokenTransaction(tokenName, 42)
	blockWithHeaders := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	testTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	assert.Nil(t, err)

	_, err = testTree.ProcessBlock(&blockWithHeaders)
	assert.Nil(t, err)

	mintTxn := testfakes.MintTokenTransaction(tokenName, 40)
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

	mintTxn2 := testfakes.MintTokenTransaction(tokenName, 1)
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

	mintTxn3 := testfakes.MintTokenTransaction(tokenName, 100)
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
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	tokenName := "testtoken"
	tokenFullName := strings.Join([]string{treeDID, tokenName}, ":")

	payload := &transactions.EstablishTokenPayload{Name: tokenName}
	payloadWrapper, err := ptypes.MarshalAny(payload)
	assert.Nil(t, err)

	txn := &transactions.Transaction{
		Type:    transactions.Transaction_ESTABLISHTOKEN,
		Payload: payloadWrapper,
	}

	blockWithHeaders := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	testTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
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
	treeDID := AddrToDid(crypto.PubkeyToAddress(treeKey.PublicKey).String())
	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	emptyTree := NewEmptyTree(treeDID, nodeStore)
	path := "some/data"
	value := "is now set"

	txn := testfakes.SetDataTransaction(path, value)
	unsignedBlock := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}
	testTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	require.Nil(t, err)

	blockWithHeaders, err := SignBlock(&unsignedBlock, treeKey)
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

	txn = testfakes.SetDataTransaction(path, value)
	unsignedBlock = chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			Height:       1,
			PreviousTip:  &testTree.Dag.Tip,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	blockWithHeaders, err = SignBlock(&unsignedBlock, treeKey)
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

	txn := testfakes.SetDataTransaction(path, value)
	unsignedBlock := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}
	testTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	require.Nil(t, err)

	blockWithHeaders, err := SignBlock(&unsignedBlock, treeKey)
	require.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	require.Nil(t, err)

	dp, err := DecodePath("/tree/data/" + path)
	require.Nil(t, err)
	resp, remain, err := testTree.Dag.Resolve(dp)
	require.Nil(t, err)
	require.Len(t, remain, 0)
	require.Equal(t, value, resp)

	txn = testfakes.SetOwnershipTransaction(keyAddr)
	unsignedBlock = chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}
	blockWithHeaders, err = SignBlock(&unsignedBlock, treeKey)
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

	tokenName := "testtoken"
	tokenFullName := strings.Join([]string{treeDID, tokenName}, ":")

	maximumAmount := uint64(50)
	height := uint64(0)

	establishTxn := testfakes.EstablishTokenTransaction(tokenName, 0)
	mintTxn := testfakes.MintTokenTransaction(tokenName, maximumAmount)
	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       height,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	testTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	assert.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	height++

	targetKey, err := crypto.GenerateKey()
	assert.Nil(t, err)
	targetTreeDID := AddrToDid(crypto.PubkeyToAddress(targetKey.PublicKey).String())

	txn := testfakes.SendTokenTransaction("1234", tokenName, 30, targetTreeDID)

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

	overSpendTxn := testfakes.SendTokenTransaction("1234", tokenName, (maximumAmount-lastSendAmount)+1, targetTreeDID)
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
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	tokenName1 := "testtoken1"

	maximumAmount := uint64(50)
	senderHeight := uint64(0)
	establishTxn := testfakes.EstablishTokenTransaction(tokenName1, 0)
	mintTxn := testfakes.MintTokenTransaction(tokenName1, maximumAmount)
	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	senderChainTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	assert.Nil(t, err)

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	// establish & mint another token to ensure we aren't relying on there only being one

	tokenName2 := "testtoken2"
	tokenFullName2 := strings.Join([]string{treeDID, tokenName2}, ":")

	establishTxn2 := testfakes.EstablishTokenTransaction(tokenName2, 0)
	mintTxn2 := testfakes.MintTokenTransaction(tokenName2, maximumAmount/2)

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
	recipientTreeDID := AddrToDid(crypto.PubkeyToAddress(recipientKey.PublicKey).String())

	sendTxn := testfakes.SendTokenTransaction("1234", tokenName2, 20, recipientTreeDID)
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

	signedBlock, err := SignBlock(sendBlockWithHeaders, key)
	require.Nil(t, err)

	headers := &StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr := crypto.PubkeyToAddress(key.PublicKey).String()
	signature, ok := headers.Signatures[sigAddr]
	require.True(t, ok)

	tokenPath := []string{"tree", "_tupelo", "tokens", tokenFullName2, TokenSendLabel, "0"}
	leafNodes, err := senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves := make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	internalSignature, err := conversion.ToInternalSignature(signature)
	require.Nil(t, err)
	receiveTxn := testfakes.ReceiveTokenTransaction("1234", senderChainTree.Dag.Tip.Bytes(), internalSignature, leaves)
	receiveBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	recipientChainTree, err := NewSignedChainTree(recipientKey.PublicKey, store)
	require.Nil(t, err)
	recipientChainTree.ChainTree.BlockValidators = append(recipientChainTree.ChainTree.BlockValidators, IsTokenRecipient)

	valid, err := recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.Nil(t, err)
	assert.True(t, valid)

	senderTree, err := senderChainTree.Tree()
	require.Nil(t, err)
	senderLedger := NewTreeLedger(senderTree, tokenFullName2)
	senderBalance, err := senderLedger.Balance()
	require.Nil(t, err)

	recipientTree, err := recipientChainTree.ChainTree.Tree()
	require.Nil(t, err)
	recipientLedger := NewTreeLedger(recipientTree, tokenFullName2)
	recipientBalance, err := recipientLedger.Balance()
	require.Nil(t, err)

	assert.Equal(t, uint64(5), senderBalance)
	assert.Equal(t, uint64(20), recipientBalance)
}

func TestReceiveTokenInvalidTip(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	tokenName1 := "testtoken1"

	maximumAmount := uint64(50)
	senderHeight := uint64(0)
	establishTxn := testfakes.EstablishTokenTransaction(tokenName1, 0)
	mintTxn := testfakes.MintTokenTransaction(tokenName1, maximumAmount)
	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	senderChainTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	assert.Nil(t, err)

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	// establish & mint another token to ensure we aren't relying on there only being one

	tokenName2 := "testtoken2"
	tokenFullName2 := strings.Join([]string{treeDID, tokenName2}, ":")

	establishTxn2 := testfakes.EstablishTokenTransaction(tokenName2, 0)
	mintTxn2 := testfakes.MintTokenTransaction(tokenName2, maximumAmount/2)
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
	recipientTreeDID := AddrToDid(crypto.PubkeyToAddress(recipientKey.PublicKey).String())

	sendTxn := testfakes.SendTokenTransaction("1234", tokenName2, 20, recipientTreeDID)
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

	signedBlock, err := SignBlock(sendBlockWithHeaders, key)
	require.Nil(t, err)

	headers := &StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr := crypto.PubkeyToAddress(key.PublicKey).String()
	signature, ok := headers.Signatures[sigAddr]
	require.True(t, ok)

	tokenPath := []string{"tree", "_tupelo", "tokens", tokenFullName2, TokenSendLabel, "0"}
	leafNodes, err := senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves := make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	otherChainTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	require.Nil(t, err)

	internalSignature, err := conversion.ToInternalSignature(signature)
	require.Nil(t, err)

	receiveTxn := testfakes.ReceiveTokenTransaction("1234", otherChainTree.Dag.Tip.Bytes(), internalSignature, leaves)
	receiveBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	recipientChainTree, err := NewSignedChainTree(recipientKey.PublicKey, store)
	require.Nil(t, err)
	recipientChainTree.ChainTree.BlockValidators = append(recipientChainTree.ChainTree.BlockValidators, IsTokenRecipient)

	valid, err := recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.NotNil(t, err)
	assert.False(t, valid)
}

func TestReceiveTokenInvalidDoubleReceive(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	tokenName1 := "testtoken1"

	maximumAmount := uint64(50)
	senderHeight := uint64(0)
	establishTxn := testfakes.EstablishTokenTransaction(tokenName1, 0)
	mintTxn := testfakes.MintTokenTransaction(tokenName1, maximumAmount)
	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	senderChainTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	assert.Nil(t, err)

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	// establish & mint another token to ensure we aren't relying on there only being one

	tokenName2 := "testtoken2"
	tokenFullName2 := strings.Join([]string{treeDID, tokenName2}, ":")

	establishTxn2 := testfakes.EstablishTokenTransaction(tokenName2, 0)
	mintTxn2 := testfakes.MintTokenTransaction(tokenName2, maximumAmount/2)
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
	recipientTreeDID := AddrToDid(crypto.PubkeyToAddress(recipientKey.PublicKey).String())

	sendTxn := testfakes.SendTokenTransaction("1234", tokenName2, 20, recipientTreeDID)
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

	signedBlock, err := SignBlock(sendBlockWithHeaders, key)
	require.Nil(t, err)

	headers := &StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr := crypto.PubkeyToAddress(key.PublicKey).String()
	signature, ok := headers.Signatures[sigAddr]
	require.True(t, ok)

	tokenPath := []string{"tree", "_tupelo", "tokens", tokenFullName2, TokenSendLabel, "0"}
	leafNodes, err := senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves := make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	internalSignature, err := conversion.ToInternalSignature(signature)
	require.Nil(t, err)

	receiveTxn := testfakes.ReceiveTokenTransaction("1234", senderChainTree.Dag.Tip.Bytes(), internalSignature, leaves)
	receiveBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	recipientChainTree, err := NewSignedChainTree(recipientKey.PublicKey, store)
	require.Nil(t, err)
	recipientChainTree.ChainTree.BlockValidators = append(recipientChainTree.ChainTree.BlockValidators, IsTokenRecipient)

	valid, err := recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.Nil(t, err)
	assert.True(t, valid)
	recipientHeight++

	// now attempt to receive a new, otherwise valid send w/ the same transaction id (which should fail)

	sendTxn2 := testfakes.SendTokenTransaction("1234", tokenName2, 2, recipientTreeDID)
	sendBlockWithHeaders = &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{sendTxn2},
		},
	}

	_, err = senderChainTree.ProcessBlock(sendBlockWithHeaders)
	assert.Nil(t, err)

	signedBlock, err = SignBlock(sendBlockWithHeaders, key)
	require.Nil(t, err)

	headers = &StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr = crypto.PubkeyToAddress(key.PublicKey).String()
	signature, ok = headers.Signatures[sigAddr]
	require.True(t, ok)

	tokenPath = []string{"tree", "_tupelo", "tokens", tokenFullName2, TokenSendLabel, "1"}
	leafNodes, err = senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves = make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	internalSignature, err = conversion.ToInternalSignature(signature)
	require.Nil(t, err)

	receiveTxn = testfakes.ReceiveTokenTransaction("1234", senderChainTree.Dag.Tip.Bytes(), internalSignature, leaves)
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
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	tokenName1 := "testtoken1"

	maximumAmount := uint64(50)
	senderHeight := uint64(0)
	establishTxn := testfakes.EstablishTokenTransaction(tokenName1, 0)
	mintTxn := testfakes.MintTokenTransaction(tokenName1, maximumAmount)
	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	senderChainTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	assert.Nil(t, err)

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	// establish & mint another token to ensure we aren't relying on there only being one

	tokenName2 := "testtoken2"
	tokenFullName2 := strings.Join([]string{treeDID, tokenName2}, ":")

	establishTxn2 := testfakes.EstablishTokenTransaction(tokenName2, 0)
	mintTxn2 := testfakes.MintTokenTransaction(tokenName2, maximumAmount/2)
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
	recipientTreeDID := AddrToDid(crypto.PubkeyToAddress(recipientKey.PublicKey).String())

	sendTxn := testfakes.SendTokenTransaction("1234", tokenName2, 20, recipientTreeDID)
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

	signedBlock, err := SignBlock(sendBlockWithHeaders, otherKey)
	require.Nil(t, err)

	headers := &StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr := crypto.PubkeyToAddress(otherKey.PublicKey).String()
	signature, ok := headers.Signatures[sigAddr]
	require.True(t, ok)

	objectID, err := senderChainTree.Id()
	require.Nil(t, err)
	signature.ObjectID = []byte(objectID)
	signature.NewTip = senderChainTree.Dag.Tip.Bytes()
	signature.PreviousTip = signedBlock.PreviousTip.Bytes()

	tokenPath := []string{"tree", "_tupelo", "tokens", tokenFullName2, TokenSendLabel, "0"}
	leafNodes, err := senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves := make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	recipientHeight := uint64(0)

	internalSignature, err := conversion.ToInternalSignature(signature)
	require.Nil(t, err)

	receiveTxn := testfakes.ReceiveTokenTransaction("1234", senderChainTree.Dag.Tip.Bytes(), internalSignature, leaves)
	receiveBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	recipientChainTree, err := NewSignedChainTree(recipientKey.PublicKey, store)
	require.Nil(t, err)

	isValidSignature := GenerateIsValidSignature(func(sig *extmsgs.Signature) (bool, error) {
		return false, nil
	})

	recipientChainTree.ChainTree.BlockValidators = append(recipientChainTree.ChainTree.BlockValidators,
		IsTokenRecipient, isValidSignature)

	valid, err := recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.Nil(t, err)
	assert.False(t, valid)
}

func TestReceiveTokenInvalidDestinationChainId(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	tokenName1 := "testtoken1"

	maximumAmount := uint64(50)
	senderHeight := uint64(0)

	establishTxn := testfakes.EstablishTokenTransaction(tokenName1, 0)
	mintTxn := testfakes.MintTokenTransaction(tokenName1, maximumAmount)
	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	senderChainTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	assert.Nil(t, err)

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	// establish & mint another token to ensure we aren't relying on there only being one

	tokenName2 := "testtoken2"
	tokenFullName2 := strings.Join([]string{treeDID, tokenName2}, ":")

	establishTxn2 := testfakes.EstablishTokenTransaction(tokenName2, 0)
	mintTxn2 := testfakes.MintTokenTransaction(tokenName2, maximumAmount/2)
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
	otherTreeDID := AddrToDid(crypto.PubkeyToAddress(otherKey.PublicKey).String())

	sendTxn := testfakes.SendTokenTransaction("1234", tokenName2, 20, otherTreeDID)
	sendBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &senderChainTree.Dag.Tip,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{sendTxn},
		},
	}

	_, err = senderChainTree.ProcessBlock(sendBlockWithHeaders)
	assert.Nil(t, err)

	signedBlock, err := SignBlock(sendBlockWithHeaders, key)
	require.Nil(t, err)

	headers := &StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr := crypto.PubkeyToAddress(key.PublicKey).String()
	signature, ok := headers.Signatures[sigAddr]
	require.True(t, ok)

	tokenPath := []string{"tree", "_tupelo", "tokens", tokenFullName2, TokenSendLabel, "0"}
	leafNodes, err := senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves := make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	recipientHeight := uint64(0)

	internalSignature, err := conversion.ToInternalSignature(signature)
	require.Nil(t, err)

	receiveTxn := testfakes.ReceiveTokenTransaction("1234", senderChainTree.Dag.Tip.Bytes(), internalSignature, leaves)
	receiveBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	recipientKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	recipientChainTree, err := NewSignedChainTree(recipientKey.PublicKey, store)
	require.Nil(t, err)
	recipientChainTree.ChainTree.BlockValidators = append(recipientChainTree.ChainTree.BlockValidators, IsTokenRecipient)

	valid, err := recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.Nil(t, err)
	assert.False(t, valid)
}

func TestReceiveTokenMismatchedSignatureTip(t *testing.T) {
	key, err := crypto.GenerateKey()
	assert.Nil(t, err)

	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeDID := AddrToDid(crypto.PubkeyToAddress(key.PublicKey).String())
	emptyTree := NewEmptyTree(treeDID, store)

	tokenName1 := "testtoken1"

	maximumAmount := uint64(50)
	senderHeight := uint64(0)
	establishTxn := testfakes.EstablishTokenTransaction(tokenName1, 0)
	mintTxn := testfakes.MintTokenTransaction(tokenName1, maximumAmount)
	blockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       senderHeight,
			Transactions: []*transactions.Transaction{establishTxn, mintTxn},
		},
	}

	senderChainTree, err := chaintree.NewChainTree(emptyTree, nil, DefaultTransactors)
	assert.Nil(t, err)

	_, err = senderChainTree.ProcessBlock(blockWithHeaders)
	assert.Nil(t, err)
	senderHeight++

	// establish & mint another token to ensure we aren't relying on there only being one

	tokenName2 := "testtoken2"
	tokenFullName2 := strings.Join([]string{treeDID, tokenName2}, ":")

	establishTxn2 := testfakes.EstablishTokenTransaction(tokenName2, 0)
	mintTxn2 := testfakes.MintTokenTransaction(tokenName2, maximumAmount/2)
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
	recipientTreeDID := AddrToDid(crypto.PubkeyToAddress(recipientKey.PublicKey).String())

	sendTxn := testfakes.SendTokenTransaction("1234", tokenName2, 20, recipientTreeDID)
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

	signedBlock, err := SignBlock(sendBlockWithHeaders, otherKey)
	require.Nil(t, err)

	headers := &StandardHeaders{}
	err = typecaster.ToType(signedBlock.Headers, headers)
	require.Nil(t, err)

	sigAddr := crypto.PubkeyToAddress(otherKey.PublicKey).String()
	signature, ok := headers.Signatures[sigAddr]
	require.True(t, ok)

	objectID, err := senderChainTree.Id()
	require.Nil(t, err)
	signature.ObjectID = []byte(objectID)
	signature.NewTip = emptyTree.Tip.Bytes() // invalid
	signature.PreviousTip = signedBlock.PreviousTip.Bytes()

	tokenPath := []string{"tree", "_tupelo", "tokens", tokenFullName2, TokenSendLabel, "0"}
	leafNodes, err := senderChainTree.Dag.NodesForPath(tokenPath)
	require.Nil(t, err)

	leaves := make([][]byte, 0)
	for _, ln := range leafNodes {
		leaves = append(leaves, ln.RawData())
	}

	recipientHeight := uint64(0)

	internalSignature, err := conversion.ToInternalSignature(signature)
	require.Nil(t, err)

	receiveTxn := testfakes.ReceiveTokenTransaction("1234", senderChainTree.Dag.Tip.Bytes(), internalSignature, leaves)
	receiveBlockWithHeaders := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       recipientHeight,
			Transactions: []*transactions.Transaction{receiveTxn},
		},
	}

	recipientChainTree, err := NewSignedChainTree(recipientKey.PublicKey, store)
	require.Nil(t, err)

	isValidSignature := GenerateIsValidSignature(func(sig *extmsgs.Signature) (bool, error) {
		return true, nil // this should get caught before it gets here; so ensure this doesn't cause false positives
	})

	recipientChainTree.ChainTree.BlockValidators = append(recipientChainTree.ChainTree.BlockValidators,
		IsTokenRecipient, isValidSignature)

	valid, err := recipientChainTree.ChainTree.ProcessBlock(receiveBlockWithHeaders)
	assert.Nil(t, err)
	assert.False(t, valid)
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
