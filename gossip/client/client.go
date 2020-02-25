package client

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-hamt-ipld"
	cbornode "github.com/ipfs/go-ipld-cbor"
	logging "github.com/ipfs/go-log"
	"github.com/opentracing/opentracing-go"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/v2/build/go/gossip"
	"github.com/quorumcontrol/messages/v2/build/go/services"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/client/pubsubinterfaces"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/types"
)

var ErrTimeout = errors.New("error timeout")
var ErrNotFound = hamt.ErrNotFound

var ErrNoRound = errors.New("no current round")
var ErrTransactionNotAccepted = errors.New("transaction not accepted")

var DefaultTimeout = 30 * time.Second

// Client represents a Tupelo client for interacting with and
// listening to ChainTree events
type Client struct {
	Group      *types.NotaryGroup
	logger     logging.EventLogger
	subscriber *roundSubscriber
	pubsub     pubsubinterfaces.Pubsubber
	store      nodestore.DagStore
}

// New instantiates a Client specific to a ChainTree/NotaryGroup. The store should probably be a bitswap peer.
// The store definitely needs access to the round confirmation, checkpoints, etc
func New(group *types.NotaryGroup, pubsub pubsubinterfaces.Pubsubber, store nodestore.DagStore) *Client {
	logger := logging.Logger("g4-client")
	subscriber := newRoundSubscriber(logger, group, pubsub, store)
	return &Client{
		Group:      group,
		logger:     logger,
		subscriber: subscriber,
		pubsub:     pubsub,
		store:      store,
	}
}

func (c *Client) Start(ctx context.Context) error {
	err := c.subscriber.start(ctx)
	if err != nil {
		return fmt.Errorf("error subscribing: %w", err)
	}

	return nil
}

func (c *Client) PlayTransactions(parentCtx context.Context, tree *consensus.SignedChainTree, treeKey *ecdsa.PrivateKey, transactions []*transactions.Transaction) (*gossip.Proof, error) {
	sp, ctx := opentracing.StartSpanFromContext(parentCtx, "client.PlayTransactions")
	defer sp.Finish()

	abr, err := c.NewAddBlockRequest(ctx, tree, treeKey, transactions)
	if err != nil {
		return nil, fmt.Errorf("error creating NewAddBlockRequest: %w", err)
	}
	c.logger.Debugf("play transactions on %s", string(abr.ObjectId))

	proof, err := c.Send(ctx, abr, DefaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("error sending tx: %w", err)
	}

	if !bytes.Equal(proof.Tip, abr.NewTip) {
		return proof, ErrTransactionNotAccepted
	}

	tipCid, err := cid.Cast(proof.Tip)
	if err != nil {
		return nil, fmt.Errorf("error casting tip cid: %v", err)
	}

	newChainTree, err := chaintree.NewChainTree(ctx, dag.NewDag(ctx, tipCid, tree.ChainTree.Dag.Store), tree.ChainTree.BlockValidators, tree.ChainTree.Transactors)
	if err != nil {
		return nil, fmt.Errorf("error creating new tree: %w", err)
	}
	tree.ChainTree = newChainTree
	return proof, nil
}

func (c *Client) GetTip(parentCtx context.Context, did string) (*gossip.Proof, error) {
	sp, ctx := opentracing.StartSpanFromContext(parentCtx, "client.GetTip")
	defer sp.Finish()

	confirmation := c.subscriber.Current()
	if confirmation == nil {
		return nil, ErrNoRound
	}
	currentRound, err := confirmation.FetchCompletedRound(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching round: %w", err)
	}
	hamtNode, err := currentRound.FetchHamt(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching hamt: %w", err)
	}

	txCID := &cid.Cid{}
	err = hamtNode.Find(ctx, did, txCID)
	if err != nil {
		if err == hamt.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error fetching tip: %w", err)
	}

	abrNode, err := c.store.Get(ctx, *txCID)
	if err != nil {
		return nil, fmt.Errorf("error getting ABR: %w", err)
	}
	abr := &services.AddBlockRequest{}
	err = cbornode.DecodeInto(abrNode.RawData(), abr)
	if err != nil {
		return nil, fmt.Errorf("error decoding ABR: %w", err)
	}

	return &gossip.Proof{
		ObjectId:          []byte(did),
		RoundConfirmation: confirmation.Value(),
		Tip:               abr.NewTip,
		AddBlockRequest:   abr,
	}, nil
}

func (c *Client) Send(parentCtx context.Context, abr *services.AddBlockRequest, timeout time.Duration) (*gossip.Proof, error) {
	sp, ctx := opentracing.StartSpanFromContext(parentCtx, "client.Send")
	defer sp.Finish()

	resp := make(chan *gossip.Proof)
	defer close(resp)

	sub, err := c.SubscribeToAbr(ctx, abr, resp)
	if err != nil {
		return nil, err
	}
	defer c.UnsubscribeFromAbr(sub)

	if err := c.SendWithoutWait(ctx, abr); err != nil {
		return nil, fmt.Errorf("error sending Tx: %w", err)
	}

	ticker := time.NewTimer(timeout)
	defer ticker.Stop()

	select {
	case proof := <-resp:
		return proof, nil
	case <-ticker.C:
		return nil, ErrTimeout
	}
}

func (c *Client) SendWithoutWait(parentCtx context.Context, abr *services.AddBlockRequest) error {
	sp, _ := opentracing.StartSpanFromContext(parentCtx, "client.SendWithoutWait")
	defer sp.Finish()

	bits, err := abr.Marshal()
	if err != nil {
		return fmt.Errorf("error marshaling: %w", err)
	}

	err = c.pubsub.Publish(c.Group.Config().TransactionTopic, bits)
	if err != nil {
		return fmt.Errorf("error publishing: %w", err)
	}

	return nil
}

func (c *Client) SubscribeToAbr(ctx context.Context, abr *services.AddBlockRequest, ch chan *gossip.Proof) (subscription, error) {
	return c.subscriber.subscribe(ctx, abr, ch)
}

func (c *Client) UnsubscribeFromAbr(s subscription) {
	c.subscriber.unsubscribe(s)
}
