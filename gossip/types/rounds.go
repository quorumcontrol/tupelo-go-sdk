package types

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-hamt-ipld"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/build/go/gossip"
	"github.com/quorumcontrol/messages/build/go/services"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/hamtwrapper"
)

func init() {
	cbornode.RegisterCborType(services.AddBlockRequest{})
	cbornode.RegisterCborType(CompletedRound{})
	cbornode.RegisterCborType(RoundConfirmation{})
}

type CompletedRound struct {
	Height     uint64
	Checkpoint cid.Cid
	State      cid.Cid
	wrapped    *cbornode.Node

	checkpoint *gossip.Checkpoint
	hamtNode   *hamt.Node
	store      nodestore.DagStore
}

func (r *CompletedRound) CID() cid.Cid {
	return r.Wrapped().Cid()
}

func (r *CompletedRound) Wrapped() *cbornode.Node {
	if r.wrapped != nil {
		return r.wrapped
	}
	sw := safewrap.SafeWrap{}
	n := sw.WrapObject(r)
	r.wrapped = n
	return n
}

func (r *CompletedRound) SetStore(store nodestore.DagStore) {
	r.store = store
}

func (r *CompletedRound) FetchCheckpoint(ctx context.Context) (*gossip.Checkpoint, error) {
	if r.checkpoint != nil {
		return r.checkpoint, nil
	}

	if r.store == nil {
		return nil, fmt.Errorf("missing a store on the completed round, use SetStore")
	}

	checkpoint := &gossip.Checkpoint{}
	checkpointNode, err := r.store.Get(ctx, r.Checkpoint)
	if err != nil {
		return nil, fmt.Errorf("error fetching checkpoint %w", err)
	}
	err = cbornode.DecodeInto(checkpointNode.RawData(), checkpoint)
	if err != nil {
		return nil, fmt.Errorf("error decoding %w", err)
	}
	r.checkpoint = checkpoint

	return checkpoint, nil
}

func (r *CompletedRound) FetchHamt(ctx context.Context) (*hamt.Node, error) {
	if r.hamtNode != nil {
		return r.hamtNode, nil
	}

	if r.store == nil {
		return nil, fmt.Errorf("missing a store on the completed round, use SetStore")
	}

	hamtStore := hamtwrapper.DagStoreToCborIpld(r.store)
	n, err := hamt.LoadNode(ctx, hamtStore, r.State, hamt.UseTreeBitWidth(5))
	if err != nil {
		return nil, fmt.Errorf("error loading hamt %w", err)
	}

	r.hamtNode = n

	return n, nil
}

type RoundConfirmation struct {
	Height         uint64
	CompletedRound cid.Cid
	Signature      *signatures.Signature
	wrapped        *cbornode.Node

	completedRound *CompletedRound
	store          nodestore.DagStore
}

func (rc *RoundConfirmation) SetStore(store nodestore.DagStore) {
	rc.store = store
}

func (rc *RoundConfirmation) FetchCompletedRound(ctx context.Context) (*CompletedRound, error) {
	if rc.completedRound != nil {
		return rc.completedRound, nil
	}

	if rc.store == nil {
		return nil, fmt.Errorf("missing a store on the round confirmation, use SetStore")
	}

	roundNode, err := rc.store.Get(ctx, rc.CompletedRound)
	if err != nil {
		return nil, fmt.Errorf("error getting node: %w", err)
	}

	completedRound := &CompletedRound{}
	err = cbornode.DecodeInto(roundNode.RawData(), completedRound)
	if err != nil {
		return nil, fmt.Errorf("error decoding: %w", err)
	}
	completedRound.SetStore(rc.store)
	rc.completedRound = completedRound

	return completedRound, nil
}

func (rc *RoundConfirmation) Data() []byte {
	return rc.Wrapped().RawData()
}

func (rc *RoundConfirmation) Wrapped() *cbornode.Node {
	if rc.wrapped != nil {
		return rc.wrapped
	}
	sw := safewrap.SafeWrap{}
	n := sw.WrapObject(rc)
	rc.wrapped = n
	return n
}
