package client

import (
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/messages/build/go/gossip"
)

func init() {
	cbornode.RegisterCborType(gossip.Proof{})
}

type ValidationNotification struct {
	AbrCid            cid.Cid
	ObjectId          string
	Tip               cid.Cid
	Checkpoint        *gossip.Checkpoint
	RoundConfirmation *gossip.RoundConfirmation
	CompletedRound    *gossip.Round

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
