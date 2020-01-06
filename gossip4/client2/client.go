package client2

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"sync"

	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-hamt-ipld"
	cbornode "github.com/ipfs/go-ipld-cbor"
	logging "github.com/ipfs/go-log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/messages/v2/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	g3types "github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip4/types"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
)

// How many times to attempt PlayTransactions before giving up.
// 10 is the library's default, but this makes it explicit.
var MaxPlayTransactionsAttempts = uint(10)

var ErrorTimeout = errors.New("error timeout")

type subscriptionCounter map[cid.Cid]uint64

// Client represents a Tupelo client for interacting with and
// listening to ChainTree events
type Client struct {
	sync.RWMutex
	Group         *g3types.NotaryGroup
	logger        logging.EventLogger
	pubsub        *pubsub.PubSub
	bitswapper    *p2p.BitswapPeer
	hamtStore     *hamt.CborIpldStore
	subscriptions subscriptionCounter
}

type Subscription struct {
	subscription *eventstream.Subscription
}

// New instantiates a Client specific to a ChainTree/NotaryGroup
func New(group *g3types.NotaryGroup, pubsub *pubsub.PubSub, bitswapper *p2p.BitswapPeer) *Client {
	hamtStore := dagStoreToCborIpld(bitswapper)

	return &Client{
		Group:         group,
		logger:        logging.Logger("client"),
		pubsub:        pubsub,
		bitswapper:    bitswapper,
		hamtStore:     hamtStore,
		subscriptions: make(subscriptionCounter),
	}
}

func (c *Client) Start(ctx context.Context) error {
	err := c.roundSubscriberStart(ctx)
	if err != nil {
		return fmt.Errorf("error subscribing: %w", err)
	}

	return nil
}

func (c *Client) subscribe(txCid cid.Cid) {
	c.Lock()
	defer c.Unlock()
	c.subscriptions[txCid]++
}

func (c *Client) roundSubscriberStart(ctx context.Context) error {
	sub, err := c.pubsub.Subscribe(c.Group.ID)
	if err != nil {
		return fmt.Errorf("error subscribing %v", err)
	}

	go func() {
		for {
			_, err := sub.Next(ctx)
			if err != nil {
				c.logger.Warningf("error getting sub message: %v", err)
				return
			}

		}
	}()
	return nil
}

func (c *Client) pubsubMessageToRoundConfirmation(ctx context.Context, msg *pubsub.Message) (*types.RoundConfirmation, error) {
	bits := msg.GetData()
	confirmation := &types.RoundConfirmation{}
	err := cbornode.DecodeInto(bits, confirmation)
	return confirmation, err
}

func (c *Client) PlayTransactions(tree *consensus.SignedChainTree, treeKey *ecdsa.PrivateKey, transactions []*transactions.Transaction) (*signatures.TreeState, error) {
	return nil, fmt.Errorf("error, undefined")
}
