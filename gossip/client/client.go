package client

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-hamt-ipld"
	cbornode "github.com/ipfs/go-ipld-cbor"
	logging "github.com/ipfs/go-log"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/v2/build/go/services"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/client/pubsubinterfaces"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/hamtwrapper"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/types"
	"time"
)

var ErrorTimeout = errors.New("error timeout")
var ErrorNotFound = hamt.ErrNotFound

var DefaultTimeout = 10 * time.Second

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

func (c *Client) PlayTransactions(ctx context.Context, tree *consensus.SignedChainTree, treeKey *ecdsa.PrivateKey, transactions []*transactions.Transaction) (*Proof, error) {
	abr, err := c.NewAddBlockRequest(ctx, tree, treeKey, transactions)
	if err != nil {
		return nil, fmt.Errorf("error creating NewAddBlockRequest: %w", err)
	}
	proof, err := c.Send(ctx, abr, DefaultTimeout)
	if err != nil {
		return nil, fmt.Errorf("error sending tx: %w", err)
	}

	newChainTree, err := chaintree.NewChainTree(ctx, dag.NewDag(ctx, proof.Tip, tree.ChainTree.Dag.Store), tree.ChainTree.BlockValidators, tree.ChainTree.Transactors)
	if err != nil {
		return nil, fmt.Errorf("error creating new tree: %w", err)
	}
	tree.ChainTree = newChainTree
	return proof, nil
}

func (c *Client) GetTip(ctx context.Context, did string) (*Proof, error) {
	confirmation := c.subscriber.Current()
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
			return nil, ErrorNotFound
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

	id, err := cid.Cast(abr.NewTip)
	if err != nil {
		return nil, fmt.Errorf("error casting tip: %w", err)
	}

	return &Proof{
		ObjectId:          did,
		RoundConfirmation: *confirmation,
		Tip:               id,
	}, nil
}

func (c *Client) Send(ctx context.Context, abr *services.AddBlockRequest, timeout time.Duration) (*Proof, error) {
	tip, err := cid.Cast(abr.NewTip)
	if err != nil {
		return nil, fmt.Errorf("error casting new tip: %w", err)
	}

	resp := make(chan *Proof)
	defer close(resp)

	sub := c.SubscribeToAbr(ctx, abr, resp)
	defer c.UnsubscribeFromAbr(sub)

	if err := c.SendWithoutWait(ctx, abr); err != nil {
		return nil, fmt.Errorf("error sending Tx: %w", err)
	}

	ticker := time.NewTimer(timeout)
	defer ticker.Stop()

	select {
	case proof := <-resp:
		proof.Tip = tip
		proof.ObjectId = string(abr.ObjectId)
		return proof, nil
	case <-ticker.C:
		return nil, ErrorTimeout
	}
}

func (c *Client) SendWithoutWait(ctx context.Context, abr *services.AddBlockRequest) error {
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

func (c *Client) SubscribeToAbr(ctx context.Context, abr *services.AddBlockRequest, ch chan *Proof) subscription {
	id := abrToHamtCID(ctx, abr)
	c.logger.Debugf("subscribing: %s", id.String())

	return c.subscriber.subscribe(id, ch)
}

func (c *Client) UnsubscribeFromAbr(s subscription) {
	c.subscriber.unsubscribe(s)
}

func abrToHamtCID(ctx context.Context, abr *services.AddBlockRequest) cid.Cid {
	underlyingStore := nodestore.MustMemoryStore(ctx)
	hamtStore := hamt.CborIpldStore{
		Blocks: hamtwrapper.NewStore(underlyingStore),
	}
	id, _ := hamtStore.Put(ctx, abr)
	return id
}
