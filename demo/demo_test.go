package demo

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	sigfuncs "github.com/quorumcontrol/tupelo-go-sdk/signatures"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConditionsDemo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group := types.NewNotaryGroup("testnotarygroup")
	validators, err := group.BlockValidators(ctx)
	require.Nil(t, err)

	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	nodeStore := nodestore.MustMemoryStore(context.TODO())

	newTree, err := consensus.NewSignedChainTree(key.PublicKey, nodeStore)
	require.Nil(t, err)

	newTree.ChainTree.BlockValidators = validators

	auths, err := newTree.Authentications()
	require.Nil(t, err)
	addr := crypto.PubkeyToAddress(key.PublicKey).String()
	require.Equal(t, auths, []string{addr})

	// Change key and test addrs
	newKey, err := crypto.GenerateKey()
	require.Nil(t, err)

	conditional := sigfuncs.EcdsaToOwnership(&newKey.PublicKey)
	conditional.Conditions = "false"

	newAddr, err := sigfuncs.Address(conditional)
	require.Nil(t, err)

	txn, err := chaintree.NewSetOwnershipTransaction([]string{newAddr.String()})
	assert.Nil(t, err)

	unsignedBlock := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  nil,
			Height:       0,
			Transactions: []*transactions.Transaction{txn},
		},
	}

	blockWithHeaders, err := consensus.SignBlock(&unsignedBlock, key)
	require.Nil(t, err)

	isValid, err := newTree.ChainTree.ProcessBlock(context.TODO(), blockWithHeaders)
	require.Nil(t, err)
	require.True(t, isValid)

	newAuths, err := newTree.Authentications()
	require.Nil(t, err)
	require.Equal(t, newAuths, []string{newAddr.String()})

	txn2, err := chaintree.NewSetDataTransaction("/test", true)
	assert.Nil(t, err)

	tip := newTree.Tip()

	unsignedBlock2 := chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			PreviousTip:  &tip,
			Height:       1,
			Transactions: []*transactions.Transaction{txn2},
		},
	}

	blockWithHeaders2, err := SignBlock(&unsignedBlock2, conditional, newKey)
	require.Nil(t, err)

	isValid2, err := newTree.ChainTree.ProcessBlock(context.TODO(), blockWithHeaders2)
	require.Nil(t, err)
	require.True(t, isValid2)

}

func SignBlock(blockWithHeaders *chaintree.BlockWithHeaders, ownership *signatures.Ownership, key *ecdsa.PrivateKey) (*chaintree.BlockWithHeaders, error) {
	hsh, err := consensus.BlockToHash(blockWithHeaders.Block)
	if err != nil {
		return nil, fmt.Errorf("error hashing block: %v", err)
	}

	sigBytes, err := crypto.Sign(hsh, key)
	if err != nil {
		return nil, fmt.Errorf("error signing: %v", err)
	}

	addr, err := sigfuncs.Address(ownership)
	if err != nil {
		return nil, fmt.Errorf("error getting address: %v", err)
	}

	sig := signatures.Signature{
		Ownership: ownership,
		Signature: sigBytes,
	}

	headers := &consensus.StandardHeaders{}
	err = typecaster.ToType(blockWithHeaders.Headers, headers)
	if err != nil {
		return nil, fmt.Errorf("error casting headers: %v", err)
	}

	if headers.Signatures == nil {
		headers.Signatures = make(consensus.SignatureMap)
	}

	headers.Signatures[addr.String()] = &sig

	var marshaledHeaders map[string]interface{}
	err = typecaster.ToType(headers, &marshaledHeaders)
	if err != nil {
		return nil, fmt.Errorf("error casting headers: %v", err)
	}

	blockWithHeaders.Headers = marshaledHeaders

	return blockWithHeaders, nil
}
