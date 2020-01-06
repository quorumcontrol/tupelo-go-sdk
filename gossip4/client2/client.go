package client2

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/AsynkronIT/protoactor-go/eventstream"
	logging "github.com/ipfs/go-log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	g3types "github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
)

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
}

type Subscription struct {
	subscription *eventstream.Subscription
}

// New instantiates a Client specific to a ChainTree/NotaryGroup
func New(group *g3types.NotaryGroup, pubsub *pubsub.PubSub, bitswapper *p2p.BitswapPeer) *Client {
	logger := logging.Logger("client")
	subscriber := newRoundSubscriber(logger, group, pubsub, bitswapper)
	return &Client{
		Group:      group,
		logger:     logger,
		subscriber: subscriber,
	}
}

func (c *Client) Start(ctx context.Context) error {
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
