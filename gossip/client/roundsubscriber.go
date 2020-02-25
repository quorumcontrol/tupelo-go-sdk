package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-hamt-ipld"
	cbornode "github.com/ipfs/go-ipld-cbor"
	logging "github.com/ipfs/go-log"
	"github.com/opentracing/opentracing-go"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/v2/build/go/gossip"
	"github.com/quorumcontrol/messages/v2/build/go/services"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/client/pubsubinterfaces"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/hamtwrapper"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/types"
	sigfuncs "github.com/quorumcontrol/tupelo-go-sdk/signatures"
)

type validationNotification struct {
	Accepted          []cid.Cid
	Checkpoint        *types.CheckpointWrapper
	RoundConfirmation *types.RoundConfirmationWrapper
	CompletedRound    *types.RoundWrapper
}

func (vn *validationNotification) includes(id cid.Cid) bool {
	for _, abrCid := range vn.Accepted {
		if abrCid.Equals(id) {
			return true
		}
	}
	return false
}

type subscription *eventstream.Subscription

type roundConflictSet map[cid.Cid]*gossip.RoundConfirmation

func isQuorum(group *types.NotaryGroup, sig *signatures.Signature) bool {
	return uint64(sigfuncs.SignerCount(sig)) >= group.QuorumCount()
}

func (rcs roundConflictSet) add(parentCtx context.Context, group *types.NotaryGroup, confirmation *gossip.RoundConfirmation) (makesQuorum bool, updated *gossip.RoundConfirmation, err error) {
	sp, ctx := opentracing.StartSpanFromContext(parentCtx, "client.rs.roundConflictSet.add")
	defer sp.Finish()

	cid, err := cid.Cast(confirmation.RoundCid)
	if err != nil {
		return false, nil, fmt.Errorf("error casting round cid: %v", err)
	}

	existing, ok := rcs[cid]
	if !ok {
		// this is the first time we're seeing the completed round,
		// just add it to the conflict set and move on
		rcs[cid] = confirmation
		return isQuorum(group, confirmation.Signature), confirmation, nil
	}

	// otherwise we've already seen a confirmation for this, let's combine the signatures
	newSig, err := sigfuncs.AggregateBLSSignatures(ctx, []*signatures.Signature{existing.Signature, confirmation.Signature})
	if err != nil {
		return false, nil, err
	}

	existing.Signature = newSig
	rcs[cid] = existing

	return isQuorum(group, existing.Signature), existing, nil
}

type conflictSetHolder map[uint64]roundConflictSet

type roundSubscriber struct {
	sync.RWMutex

	pubsub    pubsubinterfaces.Pubsubber
	dagStore  nodestore.DagStore
	hamtStore *hamt.CborIpldStore
	group     *types.NotaryGroup
	logger    logging.EventLogger

	inflight conflictSetHolder
	current  *types.RoundConfirmationWrapper

	stream *eventstream.EventStream
}

func newRoundSubscriber(logger logging.EventLogger, group *types.NotaryGroup, pubsub pubsubinterfaces.Pubsubber, store nodestore.DagStore) *roundSubscriber {
	hamtStore := hamtwrapper.DagStoreToCborIpld(store)

	return &roundSubscriber{
		pubsub:    pubsub,
		dagStore:  store,
		hamtStore: hamtStore,
		group:     group,
		inflight:  make(conflictSetHolder),
		logger:    logger,
		stream:    &eventstream.EventStream{},
	}
}

func (rs *roundSubscriber) Current() *types.RoundConfirmationWrapper {
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

func (rs *roundSubscriber) subscribe(parentCtx context.Context, abr *services.AddBlockRequest, ch chan *gossip.Proof) (subscription, error) {
	sp, ctx := opentracing.StartSpanFromContext(parentCtx, "client.rs.subscribe")

	rs.Lock()
	defer rs.Unlock()
	isDone := false
	doneCh := ctx.Done()

	abrCid, err := rs.hamtStore.Put(ctx, abr)
	if err != nil {
		rs.logger.Errorf("error decoding add block request: %v", err)
		return nil, fmt.Errorf("error decoding add block request: %v", err)
	}
	rs.logger.Debugf("subscribing: %s", abrCid.String())

	existingId := cid.Undef

	return rs.stream.Subscribe(func(evt interface{}) {
		if isDone {
			return // we're already done and we don't want to do another channel op below
		}
		defer sp.Finish()

		select {
		case <-doneCh:
			isDone = true
			return // we're already done
		default:
			// continue on, people still care
		}
		noti := evt.(*validationNotification)

		if noti.includes(abrCid) {
			p := &gossip.Proof{
				ObjectId:          abr.ObjectId,
				Tip:               abr.NewTip,
				AddBlockRequest:   abr,
				Checkpoint:        noti.Checkpoint.Value(),
				Round:             noti.CompletedRound.Value(),
				RoundConfirmation: noti.RoundConfirmation.Value(),
			}
			isDone = true
			ch <- p
			return
		}

		rs.logger.Debugf("abr %s not in accepted, falling back to hamt lookup", abrCid.String())

		// if not, then we should check to see if the object was changed underneath us
		state, err := noti.CompletedRound.FetchHamt(ctx)
		if err != nil {
			rs.logger.Errorf("error getting hamt: %v", err)
			return
		}

		newID := &cid.Cid{}

		err = state.Find(ctx, string(abr.ObjectId), newID)
		if err != nil {
			if err == hamt.ErrNotFound {
				rs.logger.Debugf("abr %s not found", abrCid.String())
				return
			}
			rs.logger.Errorf("error getting id: %v", err)
			return
		}

		// maybe we just missed a round, if so then send our proof
		if newID.Equals(abrCid) {
			rs.logger.Debug("abr %s is in the hamt, returning proof", abrCid.String())
			p := &gossip.Proof{
				ObjectId:          abr.ObjectId,
				Tip:               abr.NewTip,
				AddBlockRequest:   abr,
				Checkpoint:        noti.Checkpoint.Value(),
				Round:             noti.CompletedRound.Value(),
				RoundConfirmation: noti.RoundConfirmation.Value(),
			}
			isDone = true
			ch <- p
			return
		}

		// if we've already looked at this ID we can just ignore the rest
		if existingId.Equals(*newID) {
			rs.logger.Debugf("hamt has same ID as last time we checked")
			return // we already checked this one and can just move on
		}

		// now we're here and we have a new CID we haven't seen before - we should get the ABR and send back a proof if
		// it's higher or equal to the one we sent

		newAbr := &services.AddBlockRequest{}
		err = rs.hamtStore.Get(ctx, *newID, newAbr)
		if err != nil {
			rs.logger.Errorf("error getting abr: %v", err)
			return
		}

		if newAbr.Height >= abr.Height {
			rs.logger.Warningf("abr in hamt is newer than your subscription for %s abrTip: %s, newTip: %s", abrCid.String(), hexutil.Encode(abr.NewTip), hexutil.Encode(newAbr.NewTip))
			// the hamt changed underneath us and we need to catch up

			p := &gossip.Proof{
				ObjectId:          newAbr.ObjectId,
				Tip:               newAbr.NewTip,
				AddBlockRequest:   newAbr,
				Checkpoint:        noti.Checkpoint.Value(),
				Round:             noti.CompletedRound.Value(),
				RoundConfirmation: noti.RoundConfirmation.Value(),
			}
			isDone = true
			ch <- p
		}

	}), nil
}

func (rs *roundSubscriber) unsubscribe(sub subscription) {
	rs.Lock()
	defer rs.Unlock()
	rs.stream.Unsubscribe(sub)
}

func (rs *roundSubscriber) pubsubMessageToRoundConfirmation(ctx context.Context, msg pubsubinterfaces.Message) (*gossip.RoundConfirmation, error) {
	sp, _ := opentracing.StartSpanFromContext(ctx, "client.rs.pubsubMessageToRoundConfirmation")
	defer sp.Finish()

	bits := msg.GetData()
	confirmation := &gossip.RoundConfirmation{}
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

func (rs *roundSubscriber) handleMessage(parentCtx context.Context, msg pubsubinterfaces.Message) error {
	sp, ctx := opentracing.StartSpanFromContext(parentCtx, "client.rs.handleMessage")
	defer sp.Finish()

	confirmation, err := rs.pubsubMessageToRoundConfirmation(ctx, msg)
	if err != nil {
		return fmt.Errorf("error unmarshaling: %w", err)
	}

	if rs.current != nil && confirmation.Height <= rs.current.Height() {
		return fmt.Errorf("confirmation of height %d is less than current %d", confirmation.Height, rs.current.Height())
	}

	err = sigfuncs.RestoreBLSPublicKey(ctx, confirmation.Signature, rs.verKeys())
	if err != nil {
		return fmt.Errorf("error restoring BLS key: %w", err)
	}

	verified, err := sigfuncs.Valid(ctx, confirmation.Signature, confirmation.RoundCid, nil)
	if !verified || err != nil {
		return fmt.Errorf("signature invalid with error: %v", err)
	}

	rs.Lock()
	defer rs.Unlock()

	conflictSet, ok := rs.inflight[confirmation.Height]
	if !ok {
		conflictSet = make(roundConflictSet)
	}
	rs.logger.Debugf("checking quorum: %d", confirmation.Height)

	madeQuorum, updated, err := conflictSet.add(ctx, rs.group, confirmation)
	if err != nil {
		return fmt.Errorf("error adding to conflictset: %w", err)
	}
	rs.inflight[confirmation.Height] = conflictSet
	if madeQuorum {
		if err := rs.handleQuorum(ctx, updated); err != nil {
			return fmt.Errorf("error handling quorum: %w", err)
		}
	}

	return nil
}

func (rs *roundSubscriber) handleQuorum(ctx context.Context, confirmation *gossip.RoundConfirmation) error {
	sp := opentracing.StartSpan("client.rs.handleQuorum")
	defer sp.Finish()

	// handleQuorum expects that it's already in a lock on the roundSubscriber

	rs.logger.Debugf("handle Quorum: %d", confirmation.Height)

	wrappedConfirmation := types.WrapRoundConfirmation(confirmation)
	wrappedConfirmation.SetStore(rs.dagStore)

	rs.current = wrappedConfirmation
	for key := range rs.inflight {
		if key <= confirmation.Height {
			delete(rs.inflight, key)
		}
	}

	// fetch the completed round and confirmation here as no ops so that they are cached
	wrappedCompletedRound, err := wrappedConfirmation.FetchCompletedRound(ctx)
	if err != nil {
		return err
	}

	_, err = wrappedCompletedRound.FetchCheckpoint(ctx)
	if err != nil {
		return err
	}

	return rs.publishTxs(ctx, wrappedConfirmation)
}

func (rs *roundSubscriber) publishTxs(ctx context.Context, confirmation *types.RoundConfirmationWrapper) error {
	sp := opentracing.StartSpan("client.rs.publishTxs")
	defer sp.Finish()
	rs.logger.Debugf("publishingTxs")

	completedRound, err := confirmation.FetchCompletedRound(ctx)
	if err != nil {
		return fmt.Errorf("error fetching completed round: %w", err)
	}

	rs.logger.Debugf("getting checkpoint")
	wrappedCheckpoint, err := completedRound.FetchCheckpoint(ctx)
	if err != nil {
		return fmt.Errorf("error fetching checkpoint: %w", err)
	}
	rs.logger.Debugf("checkpoint: %v", wrappedCheckpoint.Value())

	cidBytes := wrappedCheckpoint.AddBlockRequests()
	accepted := make([]cid.Cid, len(cidBytes))
	for i, cidBytes := range wrappedCheckpoint.AddBlockRequests() {
		abrCid, err := cid.Cast(cidBytes)
		if err != nil {
			return fmt.Errorf("error casting cid: %v", err)
		}
		accepted[i] = abrCid
	}

	rs.logger.Debugf("validationNotification publish %d", confirmation.Height())
	rs.stream.Publish(&validationNotification{
		RoundConfirmation: confirmation,
		Accepted:          accepted,
		Checkpoint:        wrappedCheckpoint,
		CompletedRound:    completedRound,
	})

	return nil
}
