package types

import (
	"context"
	"fmt"

	"github.com/quorumcontrol/tupelo-go-sdk/consensus"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/messages/build/go/transactions"
)

// ValidatorGenerator is a higher order function that is used to generate a chaintree.BlockValidator that knows
// about the context it's being executed in. Specifically this is useful when the BlockValidator needs
// to know things about the notary group (like the signers) or a config (like a token necessary for transactions)
// the config stores these generators and the notary group exposes a BlockValidators function in order
// to generate validators based on the current state of the notary group.
type ValidatorGenerator func(ctx context.Context, notaryGroup *NotaryGroup) (chaintree.BlockValidatorFunc, error)

// Config is the simplest thing that could work for now
// it is just an in-memory only configuration for the notary group.
type Config struct {
	ID                  string
	TransactionToken    string
	ValidatorGenerators []ValidatorGenerator
	Transactions        map[transactions.Transaction_Type]chaintree.TransactorFunc
}

func (c *Config) blockValidators(ctx context.Context, ng *NotaryGroup) ([]chaintree.BlockValidatorFunc, error) {
	validators := make([]chaintree.BlockValidatorFunc, len(c.ValidatorGenerators))
	for i, generator := range c.ValidatorGenerators {
		validator, err := generator(ctx, ng)
		if err != nil {
			return nil, fmt.Errorf("error generating validator: %v", err)
		}
		validators[i] = validator
	}
	return validators, nil
}

// WrapBlockValidator is a convenience function when your BlockValidatorFunc does not need any state
// from the notary group or config. Currently IsOwner and IsTokenRecipient do not need any state
// and so this lets one easily wrap them.
func WrapBlockValidator(fn chaintree.BlockValidatorFunc) ValidatorGenerator {
	var validatorGenerator ValidatorGenerator = func(_ context.Context, _ *NotaryGroup) (chaintree.BlockValidatorFunc, error) {
		return fn, nil
	}
	return validatorGenerator
}

// DefaultConfig returns what we (as of this commit) use for our block validators
// GenerateIsValidSignature is ommitted in this first round because it is a higher-order
// function that needs information from a closure not known to the notary group.
// it will be special cased over in tupelo and then migrated to this format.
func DefaultConfig() *Config {
	return &Config{
		ValidatorGenerators: []ValidatorGenerator{
			WrapBlockValidator(IsOwner),
			WrapBlockValidator(IsTokenRecipient),
		},
		Transactions: consensus.DefaultTransactors,
	}
}
