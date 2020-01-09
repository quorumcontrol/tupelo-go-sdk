package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-hamt-ipld"
	cbornode "github.com/ipfs/go-ipld-cbor"
	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/hamtwrapper"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/client/pubsubinterfaces"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	sigfuncs "github.com/quorumcontrol/tupelo-go-sdk/signatures"
)

type subscription *eventstream.Subscription

type roundConflictSet map[cid.Cid]*types.RoundConfirmation

func isQuorum(group *types.NotaryGroup, sig *signatures.Signature) bool {
	return uint64(sigfuncs.SignerCount(sig)) > group.QuorumCount()
}

func (rcs roundConflictSet) add(group *types.NotaryGroup, confirmation *types.RoundConfirmation) (makesQuorum bool, updated *types.RoundConfirmation, err error) {
	ctx := context.TODO()

	existing, ok := rcs[confirmation.CompletedRound]
	if !ok {
		// this is the first time we're seeing the completed round,
		// just add it to the conflict set and move on
		rcs[confirmation.CompletedRound] = confirmation
		return isQuorum(group, confirmation.Signature), confirmation, nil
	}

	// otherwise we've already seen a confirmation for this, let's combine the signatures
	newSig, err := sigfuncs.AggregateBLSSignatures(ctx, []*signatures.Signature{existing.Signature, confirmation.Signature})
	if err != nil {
		return false, nil, err
	}

	existing.Signature = newSig
	rcs[confirmation.CompletedRound] = existing

	return isQuorum(group, existing.Signature), existing, nil
}

type conflictSetHolder map[uint64]roundConflictSet

type roundSubscriber struct {
	sync.RWMutex

	pubsub     pubsubinterfaces.Pubsubber
	bitswapper *p2p.BitswapPeer
	hamtStore  *hamt.CborIpldStore
	group      *types.NotaryGroup
	logger     logging.EventLogger

	inflight conflictSetHolder
	current  *types.RoundConfirmation

	stream *eventstream.EventStream
}

func newRoundSubscriber(logger logging.EventLogger, group *types.NotaryGroup, pubsub pubsubinterfaces.Pubsubber, bitswapper *p2p.BitswapPeer) *roundSubscriber {
	hamtStore := hamtwrapper.DagStoreToCborIpld(bitswapper)

	return &roundSubscriber{
		pubsub:     pubsub,
		bitswapper: bitswapper,
		hamtStore:  hamtStore,
		group:      group,
		inflight:   make(conflictSetHolder),
		logger:     logger,
		stream:     &eventstream.EventStream{},
	}
}

func (rs *roundSubscriber) Current() *types.RoundConfirmation {
	rs.RLock()
	defer rs.RUnlock()
	return rs.current
}

func (rs *roundSubscriber) start(ctx context.Context) error {
	sub, err := rs.pubsub.Subscribe(rs.group.ID)
	if err != nil {
		return fmt.Errorf("error subscribing %v", err)
	}

	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				rs.logger.Warningf("error getting sub message: %v", err)
				return
			}
			if err := rs.handleMessage(ctx, msg); err != nil {
				rs.logger.Warningf("error handling pubsub message: %v", err)
			}
		}
	}()
	return nil
}

func (rs *roundSubscriber) subscribe(subscriptionCID cid.Cid, ch chan *Proof) subscription {
	rs.Lock()
	defer rs.Unlock()
	return rs.stream.Subscribe(func(evt interface{}) {
		if proof := evt.(*Proof); proof.AbrCid.Equals(subscriptionCID) {
			ch <- proof
		}
	})
}

func (rs *roundSubscriber) unsubscribe(sub subscription) {
	rs.Lock()
	defer rs.Unlock()
	rs.stream.Unsubscribe(sub)
}

func (rs *roundSubscriber) pubsubMessageToRoundConfirmation(ctx context.Context, msg pubsubinterfaces.Message) (*types.RoundConfirmation, error) {
	bits := msg.GetData()
	confirmation := &types.RoundConfirmation{}
	err := cbornode.DecodeInto(bits, confirmation)
	return confirmation, err
}

// TODO: we can cache this
func (rs *roundSubscriber) verKeys() []*bls.VerKey {
	keys := make([]*bls.VerKey, len(rs.group.AllSigners()))
	for i, s := range rs.group.AllSigners() {
		keys[i] = s.VerKey
	}
	return keys
}

func (rs *roundSubscriber) handleMessage(ctx context.Context, msg pubsubinterfaces.Message) error {
	rs.logger.Debugf("handling message")

	confirmation, err := rs.pubsubMessageToRoundConfirmation(ctx, msg)
	if err != nil {
		return fmt.Errorf("error unmarshaling: %w", err)
	}

	if rs.current != nil && confirmation.Height <= rs.current.Height {
		return fmt.Errorf("confirmation of height %d is less than current %d", confirmation.Height, rs.current.Height)
	}

	err = sigfuncs.RestoreBLSPublicKey(ctx, confirmation.Signature, rs.verKeys())
	if err != nil {
		return fmt.Errorf("error restoring BLS key: %w", err)
	}

	verified, err := sigfuncs.Valid(ctx, confirmation.Signature, confirmation.CompletedRound.Bytes(), nil)
	if !verified || err != nil {
		return fmt.Errorf("signature invalid with error: %v", err)
	}

	rs.Lock()
	defer rs.Unlock()

	conflictSet, ok := rs.inflight[confirmation.Height]
	if !ok {
		conflictSet = make(roundConflictSet)
	}
	rs.logger.Debugf("checking quorum: %v", confirmation)

	madeQuorum, updated, err := conflictSet.add(rs.group, confirmation)
	if err != nil {
		return fmt.Errorf("error adding to conflictset: %w", err)
	}
	rs.inflight[confirmation.Height] = conflictSet
	if madeQuorum {
		return rs.handleQuorum(ctx, updated)
	}

	return nil
}

func (rs *roundSubscriber) handleQuorum(ctx context.Context, confirmation *types.RoundConfirmation) error {
	rs.logger.Debugf("hande Quorum: %v", confirmation)

	// handleQuorum expects that it's already in a lock on the roundSubscriber
	rs.current = confirmation
	for key := range rs.inflight {
		if key <= confirmation.Height {
			delete(rs.inflight, key)
		}
	}

	return rs.publishTxs(ctx, confirmation)
}

func (rs *roundSubscriber) publishTxs(ctx context.Context, confirmation *types.RoundConfirmation) error {
	rs.logger.Debugf("publishingTxs")
	roundNode, err := rs.bitswapper.Get(ctx, confirmation.CompletedRound)
	if err != nil {
		return err
	}

	rs.logger.Debugf("getting completed round")

	completedRound := &types.CompletedRound{}
	err = cbornode.DecodeInto(roundNode.RawData(), completedRound)
	if err != nil {
		return err
	}

	rs.logger.Debugf("getting checkpoint")

	checkpoint := &types.Checkpoint{}
	checkpointNode, err := rs.bitswapper.Get(ctx, completedRound.Checkpoint)
	if err != nil {
		return err
	}
	err = cbornode.DecodeInto(checkpointNode.RawData(), checkpoint)
	if err != nil {
		return err
	}

	rs.logger.Debugf("checkpoint: %v", checkpoint)

	for _, tx := range checkpoint.AddBlockRequests {
		rs.logger.Debugf("publishing: %s", tx.String())
		rs.stream.Publish(&Proof{
			RoundConfirmation: *confirmation,
			AbrCid:            tx,

			checkpoint:     *checkpoint,
			completedRound: *completedRound,
		})
	}
	return nil
}
