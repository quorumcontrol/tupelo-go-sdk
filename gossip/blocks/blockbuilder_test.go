package blocks

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	sigfuncs "github.com/quorumcontrol/tupelo-go-sdk/signatures"
	"github.com/stretchr/testify/require"
)

func TestNewBlockWithHeaders(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	tree, err := consensus.NewSignedChainTree(ctx, treeKey.PublicKey, nodestore.MustMemoryStore(ctx))
	require.Nil(t, err)

	t.Run("works with just a key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		block, err := NewBlockWithHeaders(ctx, tree, WithKey(treeKey))
		require.Nil(t, err)
		require.NotNil(t, block.Block)
		require.Len(t, block.Block.Transactions, 0)
	})

	t.Run("works with transactions", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		txn, err := chaintree.NewSetDataTransaction("/path", true)
		require.Nil(t, err)

		block, err := NewBlockWithHeaders(
			ctx,
			tree,
			WithKey(treeKey),
			WithTransactions([]*transactions.Transaction{txn}))
		require.Nil(t, err)
		require.NotNil(t, block.Block)
		require.Len(t, block.Block.Transactions, 1)
	})

	t.Run("works with a preimage", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		preImage := "secrets!"
		hsh := crypto.Keccak256Hash([]byte(preImage)).String()

		conditions := fmt.Sprintf(`(== (hashed-preimage) "%s")`, hsh)

		ownership := &signatures.Ownership{
			PublicKey: &signatures.PublicKey{
				Type:      signatures.PublicKey_KeyTypeSecp256k1,
				PublicKey: crypto.FromECDSAPub(&treeKey.PublicKey),
			},
			Conditions: conditions,
		}

		addr, err := sigfuncs.Address(ownership)
		require.Nil(t, err)

		ownershipTxn, err := chaintree.NewSetOwnershipTransaction([]string{addr.String()})
		require.Nil(t, err)

		setDataTxn, err := chaintree.NewSetDataTransaction("/path", true)
		require.Nil(t, err)

		block, err := NewBlockWithHeaders(
			ctx,
			tree,
			WithKey(treeKey),
			WithTransactions([]*transactions.Transaction{ownershipTxn}))
		require.Nil(t, err)
		require.NotNil(t, block.Block)
		require.Len(t, block.Block.Transactions, 1)

		// first lets play the tx on the chaintree
		newTree, valid, err := tree.ChainTree.ProcessBlockImmutable(ctx, block)
		require.Nil(t, err)
		require.True(t, valid)

		newSignedTree := consensus.NewSignedChainTreeFromChainTree(newTree)

		// now let's create a new block using the preimage

		afterOwnershipBlock, err := NewBlockWithHeaders(
			ctx,
			newSignedTree,
			WithKey(treeKey),
			WithTransactions([]*transactions.Transaction{setDataTxn}),
			WithPreImage(preImage),
			WithConditions(conditions),
		)

		require.Nil(t, err)
		require.NotNil(t, block.Block)
		require.Len(t, block.Block.Transactions, 1)
		require.NotNil(t, afterOwnershipBlock.Headers["signatures"].(map[string]interface{})[addr.String()])
	})

}
