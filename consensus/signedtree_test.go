package consensus

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/storage"
	"github.com/quorumcontrol/tupelo-go-sdk/testfakes"
	"github.com/stretchr/testify/require"
)

func TestSignedChainTree_IsGenesis(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())

	newTree, err := NewSignedChainTree(key.PublicKey, nodeStore)
	require.Nil(t, err)

	require.True(t, newTree.IsGenesis())

	txn := testfakes.SetDataTransaction("test", "value")
	unsignedBlock := testfakes.NewValidUnsignedTransactionBlock(txn)

	blockWithHeaders, err := SignBlock(&unsignedBlock, key)
	require.Nil(t, err)

	isValid, err := newTree.ChainTree.ProcessBlock(blockWithHeaders)
	require.Nil(t, err)
	require.True(t, isValid)

	require.False(t, newTree.IsGenesis())

}
