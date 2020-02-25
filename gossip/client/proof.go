package client

import (
	"github.com/ipfs/go-cid"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/types"
)

type ValidationNotification struct {
	AbrCid            cid.Cid
	ObjectId          string
	Tip               cid.Cid
	Checkpoint        *types.CheckpointWrapper
	RoundConfirmation *types.RoundConfirmationWrapper
	CompletedRound    *types.RoundWrapper
}
