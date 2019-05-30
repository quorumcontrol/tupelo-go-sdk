package types

import (
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

// Config is the simplest thing that could work for now
// it is just an in-memory only configuration for the notary group.
type Config struct {
	ID              string
	BlockValidators []chaintree.BlockValidatorFunc
	Transactions    map[transactions.Transaction_Type]chaintree.TransactorFunc
}

func DefaultConfig() *Config {
	return &Config{
		BlockValidators: []chaintree.BlockValidatorFunc{
			consensus.IsOwner,
			consensus.IsTokenRecipient,
		},
		Transactions: consensus.DefaultTransactors,
	}
}
