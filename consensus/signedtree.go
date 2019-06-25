package consensus

import (
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	cid "github.com/ipfs/go-cid"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/storage"
)

var DefaultTransactors = map[transactions.Transaction_Type]chaintree.TransactorFunc{
	transactions.Transaction_ESTABLISHTOKEN: EstablishTokenTransaction,
	transactions.Transaction_MINTTOKEN:      MintTokenTransaction,
	transactions.Transaction_SENDTOKEN:      SendTokenTransaction,
	transactions.Transaction_RECEIVETOKEN:   ReceiveTokenTransaction,
	transactions.Transaction_SETDATA:        SetDataTransaction,
	transactions.Transaction_SETOWNERSHIP:   SetOwnershipTransaction,
	transactions.Transaction_STAKE:          StakeTransaction,
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

func (sct *SignedChainTree) Authentications() ([]string, error) {
	uncastAuths, _, err := sct.ChainTree.Dag.Resolve(strings.Split("tree/"+TreePathForAuthentications, "/"))
	if err != nil {
		return nil, err
	}

	// If there are no authentications then the Chain Tree is still owned by its genesis key
	if uncastAuths == nil {
		return []string{DidToAddr(sct.MustId())}, nil
	}

	auths := make([]string, len(uncastAuths.([]interface{})))

	err = typecaster.ToType(uncastAuths, &auths)
	if err != nil {
		return nil, fmt.Errorf("error casting SignedChainTree auths: %v", err)
	}
	return auths, nil
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
