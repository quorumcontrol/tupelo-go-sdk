package blocks

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/typecaster"

	"github.com/quorumcontrol/chaintree/chaintree"

	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	sigfuncs "github.com/quorumcontrol/tupelo-go-sdk/signatures"
)

type blockBluePrint struct {
	tree         *consensus.SignedChainTree
	transactions []*transactions.Transaction
	key          *ecdsa.PrivateKey
	preImage     string
	conditions   string
}

type Option func(blueprint *blockBluePrint) error

func WithKey(key *ecdsa.PrivateKey) Option {
	return func(blueprint *blockBluePrint) error {
		blueprint.key = key
		return nil
	}
}

func WithTransactions(transactions []*transactions.Transaction) Option {
	return func(blueprint *blockBluePrint) error {
		blueprint.transactions = transactions
		return nil
	}
}

func WithConditions(conditions string) Option {
	return func(blueprint *blockBluePrint) error {
		blueprint.conditions = conditions
		return nil
	}
}

func WithPreImage(preImage string) Option {
	return func(blueprint *blockBluePrint) error {
		blueprint.preImage = preImage
		return nil
	}
}

// NewBlockWithHeaders takes a tree and a set of options and returns a blockwithheaders
func NewBlockWithHeaders(ctx context.Context, tree *consensus.SignedChainTree, opts ...Option) (*chaintree.BlockWithHeaders, error) {
	blueprint := &blockBluePrint{
		tree: tree,
	}
	for _, opt := range opts {
		err := opt(blueprint)
		if err != nil {
			return nil, fmt.Errorf("error running option: %w", err)
		}
	}
	return newBlockWithHeaders(ctx, blueprint)
}

func newBlockWithHeaders(ctx context.Context, blueprint *blockBluePrint) (*chaintree.BlockWithHeaders, error) {
	tree := blueprint.tree

	height, err := getHeight(ctx, tree)
	if err != nil {
		return nil, fmt.Errorf("error getting tree height: %v", err)
	}

	treeTip := tree.Tip()

	var blockTip *cid.Cid
	if !tree.IsGenesis() {
		blockTip = &treeTip
	}

	unsignedBlock := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			Height:       height,
			PreviousTip:  blockTip,
			Transactions: blueprint.transactions,
		},
	}

	blockWithHeaders, err := signBlock(ctx, unsignedBlock, blueprint)
	if err != nil {
		return nil, fmt.Errorf("error signing block: %w", err)
	}

	return blockWithHeaders, nil
}

func signBlock(ctx context.Context, blockWithHeaders *chaintree.BlockWithHeaders, blueprint *blockBluePrint) (*chaintree.BlockWithHeaders, error) {
	hsh, err := consensus.BlockToHash(blockWithHeaders.Block)
	if err != nil {
		return nil, fmt.Errorf("error hashing block: %v", err)
	}

	sigBytes, err := crypto.Sign(hsh, blueprint.key)
	if err != nil {
		return nil, fmt.Errorf("error signing: %v", err)
	}

	sig := signatures.Signature{
		Ownership: &signatures.Ownership{
			PublicKey: &signatures.PublicKey{
				Type:      signatures.PublicKey_KeyTypeSecp256k1,
				PublicKey: crypto.FromECDSAPub(&blueprint.key.PublicKey),
			},
			Conditions: blueprint.conditions,
		},
		Signature: sigBytes,
		PreImage:  blueprint.preImage,
	}

	addr, err := sigfuncs.Address(sig.Ownership)
	if err != nil {
		return nil, fmt.Errorf("error getting address: %w", err)
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

func getHeight(ctx context.Context, tree *consensus.SignedChainTree) (uint64, error) {
	ct := tree.ChainTree

	unmarshaledRoot, err := ct.Dag.Get(ctx, ct.Dag.Tip)
	if unmarshaledRoot == nil || err != nil {
		return 0, fmt.Errorf("error, missing root: %v", err)
	}

	root := &chaintree.RootNode{}

	err = cbornode.DecodeInto(unmarshaledRoot.RawData(), root)
	if err != nil {
		return 0, fmt.Errorf("error decoding root: %v", err)
	}

	var height uint64
	if tree.IsGenesis() {
		height = 0
	} else {
		height = root.Height + 1
	}

	return height, nil
}
