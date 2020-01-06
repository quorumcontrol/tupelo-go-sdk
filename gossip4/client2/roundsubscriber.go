package client2

import (
	"context"
	"fmt"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-hamt-ipld"
	cbornode "github.com/ipfs/go-ipld-cbor"
	logging "github.com/ipfs/go-log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	g3types "github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip4/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	sigfuncs "github.com/quorumcontrol/tupelo-go-sdk/signatures"
)

type subscriptionCounter map[cid.Cid]uint64

type roundConflictSet map[cid.Cid]*types.RoundConfirmation

func isQuorum(group *g3types.NotaryGroup, sig *signatures.Signature) bool {
	return uint64(len(sig.Signers)) > group.QuorumCount()
}

func (rcs roundConflictSet) add(group *g3types.NotaryGroup, confirmation *types.RoundConfirmation) (makesQuorum bool, updated *types.RoundConfirmation, err error) {
	existing, ok := rcs[confirmation.CompletedRound]
	if !ok {
		// this is the first time we're seeing the completed round,
		// just add it to the conflict set and move on
		rcs[confirmation.CompletedRound] = confirmation
		return isQuorum(group, confirmation.Signature), existing, nil
	}

	// otherwise we've already seen a confirmation for this, let's combine the signatures
	newSig, err := sigfuncs.AggregateBLSSignatures([]*signatures.Signature{existing.Signature, confirmation.Signature})
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

	pubsub        *pubsub.PubSub
	bitswapper    *p2p.BitswapPeer
	hamtStore     *hamt.CborIpldStore
	subscriptions subscriptionCounter
	group         *g3types.NotaryGroup
	logger        logging.EventLogger

	inflight conflictSetHolder
	current  *types.RoundConfirmation
}

func newRoundSubscriber(logger logging.EventLogger, group *g3types.NotaryGroup, pubsub *pubsub.PubSub, bitswapper *p2p.BitswapPeer) *roundSubscriber {
	hamtStore := dagStoreToCborIpld(bitswapper)

	return &roundSubscriber{
		pubsub:        pubsub,
		bitswapper:    bitswapper,
		hamtStore:     hamtStore,
		subscriptions: make(subscriptionCounter),
		group:         group,
		inflight:      make(conflictSetHolder),
		logger:        logger,
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

func (rs *roundSubscriber) subscribe(txCid cid.Cid) {
	rs.Lock()
	defer rs.Unlock()
	rs.subscriptions[txCid]++
}

func (rs *roundSubscriber) pubsubMessageToRoundConfirmation(ctx context.Context, msg *pubsub.Message) (*types.RoundConfirmation, error) {
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

func (rs *roundSubscriber) handleMessage(ctx context.Context, msg *pubsub.Message) error {
	confirmation, err := rs.pubsubMessageToRoundConfirmation(ctx, msg)
	if err != nil {
		return fmt.Errorf("error unmarshaling: %w", err)
	}

	if rs.current != nil && confirmation.Height <= rs.current.Height {
		return fmt.Errorf("confirmation of height %d is less than current %d", confirmation.Height, rs.current.Height)
	}

	sigfuncs.RestoreBLSPublicKey(confirmation.Signature, rs.verKeys())

	verified, err := sigfuncs.Valid(confirmation.Signature, confirmation.CompletedRound.Bytes(), nil)
	if !verified || err != nil {
		return fmt.Errorf("signature invalid with error: %v", err)
	}

	rs.Lock()
	defer rs.Unlock()

	conflictSet, ok := rs.inflight[confirmation.Height]
	if !ok {
		conflictSet = make(roundConflictSet)
	}

	madeQuorum, updated, err := conflictSet.add(rs.group, confirmation)
	if madeQuorum {
		return rs.handleQuorum(ctx, updated)
	}

	return nil
}

func (rs *roundSubscriber) handleQuorum(ctx context.Context, confirmation *types.RoundConfirmation) error {
	// handleQuorum expects that it's already in a lock on the roundSubscriber
	rs.current = confirmation
	for key := range rs.inflight {
		if key <= confirmation.Height {
			delete(rs.inflight, key)
		}
	}

	return nil
}
