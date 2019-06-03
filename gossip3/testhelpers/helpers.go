package testhelpers

import (
	"github.com/quorumcontrol/messages/build/go/services"
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/storage"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func NewValidTransaction(t testing.TB) services.AddBlockRequest {
	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)

	return NewValidTransactionWithKey(t, treeKey)
}

func NewValidTransactionWithKey(t testing.TB, treeKey *ecdsa.PrivateKey) services.AddBlockRequest {
	return NewValidTransactionWithPathAndValue(t, treeKey, "down/in/the/thing", "hi")
}

func NewValidTransactionWithPathAndValue(t testing.TB, treeKey *ecdsa.PrivateKey, path, value string) services.AddBlockRequest {
	sw := safewrap.SafeWrap{}

	txn, err := chaintree.NewSetDataTransaction(path, value)
	require.Nil(t, err)

	unsignedBlock := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	treeDID := consensus.AddrToDid(crypto.PubkeyToAddress(treeKey.PublicKey).String())

	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	emptyTree := consensus.NewEmptyTree(treeDID, nodeStore)
	emptyTip := emptyTree.Tip
	testTree, err := chaintree.NewChainTree(emptyTree, nil, consensus.DefaultTransactors)
	require.Nil(t, err)

	blockWithHeaders, err := consensus.SignBlock(&unsignedBlock, treeKey)
	require.Nil(t, err)

	_, err = testTree.ProcessBlock(blockWithHeaders)
	require.Nil(t, err)
	nodes := DagToByteNodes(t, emptyTree)

	bits := sw.WrapObject(blockWithHeaders).RawData()
	require.Nil(t, sw.Err)

	return services.AddBlockRequest{
		PreviousTip: emptyTip.Bytes(),
		Height:      blockWithHeaders.Height,
		NewTip:      testTree.Dag.Tip.Bytes(),
		Payload:     bits,
		State:       nodes,
		ObjectId:    []byte(treeDID),
	}
}

func DagToByteNodes(t testing.TB, dagTree *dag.Dag) [][]byte {
	cborNodes, err := dagTree.Nodes()
	require.Nil(t, err)
	nodes := make([][]byte, len(cborNodes))
	for i, node := range cborNodes {
		nodes[i] = node.RawData()
	}
	return nodes
}
