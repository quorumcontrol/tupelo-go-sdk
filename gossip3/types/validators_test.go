package types

import (
	"github.com/stretchr/testify/require"
	"context"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/chaintree/chaintree"
	"testing"
)


func TestBurnValidator(t *testing.T) {
	ctx,cancel := context.WithCancel(context.Background())
	defer cancel()

	config := DefaultConfig()

	config.ID = "TestBurnValidator"
	config.TransactionToken = "did:atest/ink"
	config.ValidatorGenerators = append(config.ValidatorGenerators, HasBurnGenerator)

	ng := NewNotaryGroupFromConfig(config)
	validator,err := HasBurnGenerator(ctx, ng)
	require.Nil(t,err)
	
	blockWithTxs := func(txs []*transactions.Transaction) *chaintree.BlockWithHeaders {
		return &chaintree.BlockWithHeaders{
			Block: chaintree.Block{
				PreviousTip:  nil,
				Height:       0,
				Transactions: txs,
			},
		}
	}

	t.Run("with a valid burn", func(t *testing.T) {
		blockWithHeaders := blockWithTxs([]*transactions.Transaction{
			&transactions.Transaction{
				Type: transactions.Transaction_SENDTOKEN,
				SendTokenPayload: &transactions.SendTokenPayload{
					Name: config.TransactionToken,
					Amount: 1,
					Destination: "",
				},
			}})
		isValid,err := validator(nil, blockWithHeaders)
		require.Nil(t,err)
		require.True(t, isValid)
	})

}