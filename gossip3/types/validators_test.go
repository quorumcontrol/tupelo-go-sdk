package types

import (
	"context"
	"testing"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/stretchr/testify/require"
)

func TestBurnValidator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := DefaultConfig()

	config.ID = "TestBurnValidator"
	config.TransactionToken = "did:atest/ink"
	config.BurnAmount = 1
	config.ValidatorGenerators = append(config.ValidatorGenerators, HasBurnGenerator)

	ng := NewNotaryGroupFromConfig(config)
	validator, err := HasBurnGenerator(ctx, ng)
	require.Nil(t, err)

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
					Name:        config.TransactionToken,
					Amount:      1,
					Destination: "",
				},
			}})
		isValid, err := validator(nil, blockWithHeaders)
		require.Nil(t, err)
		require.True(t, isValid)
	})

	t.Run("with no burn", func(t *testing.T) {
		blockWithHeaders := blockWithTxs(nil)
		isValid, err := validator(nil, blockWithHeaders)
		require.Nil(t, err)
		require.False(t, isValid)
	})

	t.Run("with 0 value burn", func(t *testing.T) {
		blockWithHeaders := blockWithTxs([]*transactions.Transaction{
			&transactions.Transaction{
				Type: transactions.Transaction_SENDTOKEN,
				SendTokenPayload: &transactions.SendTokenPayload{
					Name:        config.TransactionToken,
					Amount:      0,
					Destination: "",
				},
			}})
		isValid, err := validator(nil, blockWithHeaders)
		require.Nil(t, err)
		require.False(t, isValid)
	})

	t.Run("with wrong token burn", func(t *testing.T) {
		blockWithHeaders := blockWithTxs([]*transactions.Transaction{
			&transactions.Transaction{
				Type: transactions.Transaction_SENDTOKEN,
				SendTokenPayload: &transactions.SendTokenPayload{
					Name:        "different token",
					Amount:      100,
					Destination: "",
				},
			}})
		isValid, err := validator(nil, blockWithHeaders)
		require.Nil(t, err)
		require.False(t, isValid)
	})

	t.Run("with a configured burn amount", func(t *testing.T) {
		config.BurnAmount = 100

		defer func() { config.BurnAmount = 1 }()

		validator, err := HasBurnGenerator(ctx, ng)
		require.Nil(t, err)

		blockWithHeaders := blockWithTxs([]*transactions.Transaction{
			&transactions.Transaction{
				Type: transactions.Transaction_SENDTOKEN,
				SendTokenPayload: &transactions.SendTokenPayload{
					Name:        config.TransactionToken,
					Amount:      99,
					Destination: "",
				},
			}})
		isValid, err := validator(nil, blockWithHeaders)
		require.Nil(t, err)
		require.False(t, isValid)

		blockWithHeaders = blockWithTxs([]*transactions.Transaction{
			&transactions.Transaction{
				Type: transactions.Transaction_SENDTOKEN,
				SendTokenPayload: &transactions.SendTokenPayload{
					Name:        config.TransactionToken,
					Amount:      100,
					Destination: "",
				},
			}})
		isValid, err = validator(nil, blockWithHeaders)
		require.Nil(t, err)
		require.True(t, isValid)
	})

}
