package client

import (
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/messages/build/go/gossip"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/types"
)

func init() {
	cbornode.RegisterCborType(Proof{})
}

type Proof struct {
	RoundConfirmation types.RoundConfirmation
	AbrCid            cid.Cid
	ObjectId          string
	Tip               cid.Cid

	completedRound types.CompletedRound
	checkpoint     gossip.Checkpoint

	// dag store and hamt?
}
