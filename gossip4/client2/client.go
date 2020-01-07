package client2

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"time"

	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/v2/build/go/services"

	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-hamt-ipld"
	logging "github.com/ipfs/go-log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	g3types "github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip4/hamtwrapper"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
)

const transactionTopic = "g4-transactions"

// How many times to attempt PlayTransactions before giving up.
// 10 is the library's default, but this makes it explicit.
var MaxPlayTransactionsAttempts = uint(10)

var ErrorTimeout = errors.New("error timeout")

// Client represents a Tupelo client for interacting with and
// listening to ChainTree events
type Client struct {
	Group      *g3types.NotaryGroup
	logger     logging.EventLogger
	subscriber *roundSubscriber
	pubsub     *pubsub.PubSub
}

type Subscription struct {
	subscription *eventstream.Subscription
}

// New instantiates a Client specific to a ChainTree/NotaryGroup
func New(group *g3types.NotaryGroup, pubsub *pubsub.PubSub, bitswapper *p2p.BitswapPeer) *Client {
	logger := logging.Logger("g4-client")
	subscriber := newRoundSubscriber(logger, group, pubsub, bitswapper)
	return &Client{
		Group:      group,
		logger:     logger,
		subscriber: subscriber,
		pubsub:     pubsub,
	}
}

func (c *Client) Start(ctx context.Context) error {
	c.pubsub.Subscribe(transactionTopic) // TODO: do we need this?

	err := c.roundSubscriberStart(ctx)
	if err != nil {
		return fmt.Errorf("error subscribing: %w", err)
	}

	return nil
}

func (c *Client) roundSubscriberStart(ctx context.Context) error {
	return c.subscriber.start(ctx)
}

func (c *Client) PlayTransactions(tree *consensus.SignedChainTree, treeKey *ecdsa.PrivateKey, transactions []*transactions.Transaction) (*signatures.TreeState, error) {
	return nil, fmt.Errorf("error, undefined")
}

func (c *Client) Send(ctx context.Context, abr *services.AddBlockRequest, timeout time.Duration) error {
	bits, err := abr.Marshal()
	if err != nil {
		return fmt.Errorf("error marshaling: %w", err)
	}

	resp := make(chan error)

	id := abrToHamtCID(ctx, abr)
	c.logger.Debugf("sending: %s", id.String())

	sub := c.subscriber.subscribe(id, resp)
	defer c.subscriber.unsubscribe(sub)

	err = c.pubsub.Publish(transactionTopic, bits)
	if err != nil {
		return fmt.Errorf("error publishing: %w", err)
	}
	ticker := time.NewTimer(timeout)
	defer ticker.Stop()

	select {
	case err := <-resp:
		return err
	case <-ticker.C:
		return ErrorTimeout
	}
}

func abrToHamtCID(ctx context.Context, abr *services.AddBlockRequest) cid.Cid {
	underlyingStore := nodestore.MustMemoryStore(ctx)
	hamtStore := hamt.CborIpldStore{
		Blocks: hamtwrapper.NewStore(underlyingStore),
	}
	id, _ := hamtStore.Put(ctx, abr)
	return id
}
