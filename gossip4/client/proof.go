package client

import (
	"github.com/ipfs/go-cid"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip4/types"
)

type Proof struct {
	RoundConfirmation types.RoundConfirmation
	AbrCid            cid.Cid
	ObjectId          string
	Tip               cid.Cid

	completedRound types.CompletedRound
	checkpoint     types.Checkpoint

	// dag store and hamt?
}
