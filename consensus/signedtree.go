package consensus

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	cid "github.com/ipfs/go-cid"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/transactions"
	"github.com/quorumcontrol/storage"
)

const (
	TransactionTypeEstablishToken = "ESTABLISH_TOKEN"
	TransactionTypeMintToken      = "MINT_TOKEN"
	TransactionTypeSetData        = "SET_DATA"
	TransactionTypeSetOwnership   = "SET_OWNERSHIP"
	TransactionTypeStake          = "STAKE"
)

var DefaultTransactors = map[transactions.TransactionType]chaintree.TransactorFunc{
	transactions.TransactionType_EstablishToken: EstablishTokenTransaction,
	transactions.TransactionType_MintToken:      MintTokenTransaction,
	transactions.TransactionType_SetData:        SetDataTransaction,
	transactions.TransactionType_SetOwnership:   SetOwnershipTransaction,
	transactions.TransactionType_Stake:          StakeTransaction,
}

type SignedChainTree struct {
	ChainTree  *chaintree.ChainTree
	Signatures SignatureMap
}

func (sct *SignedChainTree) Id() (string, error) {
	return sct.ChainTree.Id()
}

func (sct *SignedChainTree) MustId() string {
	id, err := sct.ChainTree.Id()
	if err != nil {
		log.Error("error getting id from chaintree: %v", id)
	}
	return id
}

func (sct *SignedChainTree) Tip() cid.Cid {
	return sct.ChainTree.Dag.Tip
}

func (sct *SignedChainTree) IsGenesis() bool {
	store := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	newEmpty := NewEmptyTree(sct.MustId(), store)
	return newEmpty.Tip.Equals(sct.Tip())
}

func NewSignedChainTree(key ecdsa.PublicKey, nodeStore nodestore.NodeStore) (*SignedChainTree, error) {
	did := EcdsaPubkeyToDid(key)

	tree, err := chaintree.NewChainTree(
		NewEmptyTree(did, nodeStore),
		nil,
		DefaultTransactors,
	)

	if err != nil {
		return nil, fmt.Errorf("error creating tree: %v", err)
	}

	return &SignedChainTree{
		ChainTree:  tree,
		Signatures: make(SignatureMap),
	}, nil
}
