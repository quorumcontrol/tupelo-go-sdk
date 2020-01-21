package client

import (
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/messages/v2/build/go/gossip"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/types"
)

func init() {
	cbornode.RegisterCborType(gossip.Proof{})
}

type ValidationNotification struct {
	AbrCid            cid.Cid
	ObjectId          string
	Tip               cid.Cid
	Checkpoint        *types.CheckpointWrapper
	RoundConfirmation *types.RoundConfirmationWrapper
	CompletedRound    *types.RoundWrapper

	// dag store and hamt?
}

// func (p *Proof) Prove(tx *transactions.Transaction) error {
//	// Transaction
//	// AddBlockRequest
//	// Checkpoint
//	// Round
//	// RoundConfirmation
//	return nil
// }
