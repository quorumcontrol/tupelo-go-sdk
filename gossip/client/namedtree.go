package client

import (
	"context"
	"crypto/ecdsa"
	"strings"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

type Generator struct {
	Namespace string
	Client    *Client
}

type NamedChainTree struct {
	Name      string
	ChainTree *consensus.SignedChainTree
	Client    *Client

	genesisKey *ecdsa.PrivateKey
	nodeStore  nodestore.DagStore
	owners     []string
}

type NamedChainTreeOptions struct {
	Name              string
	ObjectStorageType string
	Client            *Client
	NodeStore         nodestore.DagStore
	Owners            []string
}

// GenesisKey creates a new named key creation key based on the supplied name.
// It lower-cases the name first to ensure that chaintree names are case
// insensitive.
func (g *Generator) GenesisKey(name string) (*ecdsa.PrivateKey, error) {
	// TODO: If we ever want case sensitivity, consider adding a bool flag
	// to the Generator or adding a new fun.
	lowerCased := strings.ToLower(name)
	return consensus.PassPhraseKey([]byte(lowerCased), []byte(g.Namespace))
}

func (g *Generator) New(ctx context.Context, opts *NamedChainTreeOptions) (*NamedChainTree, error) {
	gKey, err := g.GenesisKey(opts.Name)
	if err != nil {
		return nil, err
	}

	chainTree, err := consensus.NewSignedChainTree(ctx, gKey.PublicKey, opts.NodeStore)
	if err != nil {
		return nil, err
	}

	setOwnershipTxn, err := chaintree.NewSetOwnershipTransaction(opts.Owners)
	if err != nil {
		return nil, err
	}

	_, err = opts.Client.PlayTransactions(ctx, chainTree, gKey, []*transactions.Transaction{setOwnershipTxn})
	if err != nil {
		return nil, err
	}

	return &NamedChainTree{
		Name:       opts.Name,
		ChainTree:  chainTree,
		genesisKey: gKey,
		Client:     opts.Client,
		nodeStore:  opts.NodeStore,
		owners:     opts.Owners,
	}, nil
}

func (g *Generator) Did(name string) (string, error) {
	gKey, err := g.GenesisKey(name)
	if err != nil {
		return "", err
	}

	return consensus.EcdsaPubkeyToDid(gKey.PublicKey), nil
}

func (g *Generator) Find(ctx context.Context, name string) (*NamedChainTree, error) {
	did, err := g.Did(name)
	if err != nil {
		return nil, err
	}

	chainTree, err := g.Client.GetLatest(ctx, did)
	if err == ErrNotFound {
		return nil, ErrNotFound
	}

	gKey, err := g.GenesisKey(name)
	if err != nil {
		return nil, err
	}

	return &NamedChainTree{
		Name:       name,
		ChainTree:  chainTree,
		genesisKey: gKey,
		Client:     g.Client,
	}, nil
}

func (t *NamedChainTree) Did() string {
	return consensus.EcdsaPubkeyToDid(t.genesisKey.PublicKey)
}
