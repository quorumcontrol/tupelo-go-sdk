package types

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"

	"github.com/BurntSushi/toml"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

func init() {
	for enum, fn := range consensus.DefaultTransactors {
		mustRegisterTransactor(transactions.Transaction_Type_name[int32(enum)], fn)
	}
}

// transactorRegistry allows for registering your functions to human-readable srings
var transactorRegistry = make(map[string]chaintree.TransactorFunc)

// RegisterTransactor is used to make transactors available for the human-readable configs
func RegisterTransactor(name string, fn chaintree.TransactorFunc) error {
	_, ok := transactions.Transaction_Type_value[name]
	if !ok {
		return fmt.Errorf("error: you must specify a name that is specified in transactions protobufs")
	}
	_, ok = transactorRegistry[name]
	if ok {
		return fmt.Errorf("error: %s already exists in the transactor registry", name)
	}
	transactorRegistry[name] = fn
	return nil
}

func mustRegisterTransactor(name string, fn chaintree.TransactorFunc) {
	err := RegisterTransactor(name, fn)
	if err != nil {
		panic(err)
	}
}

// validatorGeneratorRegistry is used for human-readable configs to specify what validators
// should be used for a notary group.
var validatorGeneratorRegistry = make(map[string]ValidatorGenerator)

// RegisterValidatorGenerator registers your validator generator with a human-readable name
// so that it can be specified in the on-disk configs.
func RegisterValidatorGenerator(name string, fn ValidatorGenerator) error {
	_, ok := validatorGeneratorRegistry[name]
	if ok {
		return fmt.Errorf("error: %s already exists in the validator registry", name)
	}
	validatorGeneratorRegistry[name] = fn
	return nil
}

func mustRegisterValidatorGenerator(name string, fn ValidatorGenerator) {
	err := RegisterValidatorGenerator(name, fn)
	if err != nil {
		panic(err)
	}
}

type PublicKeySet struct {
	VerKey  *bls.VerKey
	DestKey *ecdsa.PublicKey
}

type HumanPublicKeySet struct {
	VerKeyHex  string
	DestKeyHex string
}

func (hpubset *HumanPublicKeySet) ToPublicKeySet() (pubset PublicKeySet, err error) {
	blsBits, err := hexutil.Decode(hpubset.VerKeyHex)
	if err != nil {
		return pubset, fmt.Errorf("error decoding verkey: %v", err)
	}
	ecdsaBits, err := hexutil.Decode(hpubset.DestKeyHex)
	if err != nil {
		return pubset, fmt.Errorf("error decoding destkey: %v", err)
	}

	ecdsaPub, err := crypto.UnmarshalPubkey(ecdsaBits)
	if err != nil {
		return pubset, fmt.Errorf("couldn't unmarshal ECDSA pub key: %v", err)
	}

	verKey := bls.BytesToVerKey(blsBits)

	return PublicKeySet{
		DestKey: ecdsaPub,
		VerKey:  verKey,
	}, nil
}

// HumanConfig is used for parsing an ondisk configuration into the application-used Config
// struct.
type HumanConfig struct {
	// See Config
	ID string
	// See Config
	TransactionToken string
	// See Config
	BurnAmount uint64
	// See Config
	TransactionTopic string
	// See Config
	CommitTopic string
	// ValidatorGenerators is an array of strings representing which Validators
	// should be run as part of this notary group. The validators must be registered with
	// RegisterValidatorGenerator. The built in validators (ISOWNER, ISRECIPIENT, HASBURN at time
	// of this comment) are already registered and the default is ["ISOWNER", "ISRECIPIENT"] if no
	// generators are specified.
	ValidatorGenerators []string
	// Transactions is a slice of strings representing the Protobuf TransactionType allowed
	// in this notary group. Default transactions (consensus.DefaultTransactors) are pre-registered
	// otherwise you must register a transaction using RegisterTransactor. Default if unspecified
	// is consensus.DefaultTransactors.
	Transactions []string
	// Signers is a slice of hex BLS VerKeys and ecdsa public keys (for libp2p)
	Signers []HumanPublicKeySet
}

func HumanConfigToConfig(hc *HumanConfig) (*Config, error) {
	defaults := DefaultConfig()
	c := &Config{
		ID:               hc.ID,
		TransactionToken: hc.TransactionToken,
		BurnAmount:       hc.BurnAmount,
		TransactionTopic: hc.TransactionTopic,
		CommitTopic:      hc.CommitTopic,
	}

	if c.ID == "" {
		return nil, fmt.Errorf("error ID cannot be nil")
	}

	if c.TransactionToken == "" {
		c.TransactionToken = defaults.TransactionToken // at this moment we expect default to be "" too
	}

	if c.BurnAmount == 0 {
		c.BurnAmount = defaults.BurnAmount // at this moment, we expect default to be 0 too
	}

	if c.TransactionTopic == "" {
		c.TransactionTopic = defaults.TransactionTopic
	}

	if c.CommitTopic == "" {
		c.CommitTopic = defaults.CommitTopic
	}

	if len(hc.ValidatorGenerators) > 0 {
		for _, generatorName := range hc.ValidatorGenerators {
			generator, ok := validatorGeneratorRegistry[generatorName]
			if !ok {
				return nil, fmt.Errorf("error no generator of name %s", generatorName)
			}
			c.ValidatorGenerators = append(c.ValidatorGenerators, generator)
		}
	} else {
		c.ValidatorGenerators = defaults.ValidatorGenerators
	}

	if len(hc.Transactions) > 0 {
		c.Transactions = make(map[transactions.Transaction_Type]chaintree.TransactorFunc)
		for _, transactionName := range hc.Transactions {
			enum, ok := transactions.Transaction_Type_value[transactionName]
			if !ok {
				return nil, fmt.Errorf("error: you must specify a name that is specified in transactions protobufs")
			}
			fn, ok := transactorRegistry[transactionName]
			if !ok {
				return nil, fmt.Errorf("error: you must specify a name that is registered using RegisterTransactor")
			}
			c.Transactions[transactions.Transaction_Type(enum)] = fn
		}
	} else {
		c.Transactions = defaults.Transactions
	}

	signers := make([]PublicKeySet, len(hc.Signers))
	for i, humanPub := range hc.Signers {
		pub, err := humanPub.ToPublicKeySet()
		if err != nil {
			return nil, fmt.Errorf("error getting signer from human: %v", err)
		}
		signers[i] = pub
	}
	c.Signers = signers

	return c, nil
}

// TomlToConfig will load a notary group config from a toml string
// Generally these configs are nested and so you will rarely need
// to use this function, it's more to validate example files
// and to use in tests.
func TomlToConfig(tomlBytes string) (*Config, error) {
	var hc HumanConfig
	_, err := toml.Decode(tomlBytes, &hc)
	if err != nil {
		return nil, fmt.Errorf("error decoding toml: %v", err)
	}
	return HumanConfigToConfig(&hc)
}
