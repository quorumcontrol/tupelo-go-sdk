package client

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	format "github.com/ipfs/go-ipld-format"

	"github.com/quorumcontrol/messages/build/go/services"
	"github.com/quorumcontrol/messages/build/go/signatures"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/eventstream"
	"github.com/avast/retry-go"
	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"go.uber.org/zap"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/safewrap"

	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
)

// How many times to attempt PlayTransactions before giving up.
// 10 is the library's default, but this makes it explicit.
const MaxPlayTransactionsAttempts = uint(10)

const ErrorTimeout = "error timeout"

// Client represents a Tupelo client for interacting with and
// listening to ChainTree events
type Client struct {
	Group      *types.NotaryGroup
	TreeDID    string
	log        *zap.SugaredLogger
	pubsub     remote.PubSub
	subscriber *actor.PID
	cache      *lru.Cache
	stream     *eventstream.EventStream
}

type Subscription struct {
	subscription *eventstream.Subscription
}

// New instantiates a Client specific to a ChainTree/NotaryGroup
func New(group *types.NotaryGroup, treeDid string, pubsub remote.PubSub) *Client {
	cache, err := lru.New(10000)
	if err != nil {
		panic(fmt.Errorf("error generating LRU: %v", err))
	}
	return &Client{
		Group:   group,
		TreeDID: treeDid,
		log:     middleware.Log.Named("client-" + treeDid),
		pubsub:  pubsub,
		cache:   cache,
		stream:  &eventstream.EventStream{},
	}
}

func (c *Client) Listen() {
	if c.alreadyListening() {
		return
	}
	c.subscriber = actor.EmptyRootContext.SpawnPrefix(actor.PropsFromFunc(c.subscriptionReceive), c.TreeDID+"-subscriber")
}

func (c *Client) alreadyListening() bool {
	return c.subscriber != nil
}

// Stop stops a Client.
func (c *Client) Stop() {
	if c.alreadyListening() {
		c.subscriber.Stop()
		c.subscriber = nil
	}
}

func (c *Client) subscriptionReceive(actorContext actor.Context) {
	switch msg := actorContext.Message().(type) {
	case *actor.Started:
		_, err := actorContext.SpawnNamed(c.pubsub.NewSubscriberProps(c.TreeDID), "pubsub")
		if err != nil {
			panic(fmt.Errorf("error spawning pubsub: %v", err))
		}
	case *signatures.TreeState:
		if msg.Signature == nil {
			c.log.Errorw("received signatures.CurrentState message without signature")
			return
		}

		heightString := strconv.FormatUint(msg.Height, 10)
		//TODO: this needs to check the validity of the signature
		existed, _ := c.cache.ContainsOrAdd(heightString, msg)
		if !existed {
			c.log.Debugw("publishing current state", "objectID", string(msg.ObjectId),
				"height", heightString)
			c.stream.Publish(msg)
		}
	default:
		c.log.Debugw("unknown message received", "type", reflect.TypeOf(msg).String())
	}
}

// TipRequest requests the tip of a chain tree.
func (c *Client) TipRequest() (*signatures.TreeState, error) {
	var attemptNo int
	var res interface{}
	err := retry.Do(
		func() error {
			var err error
			timeout := time.Duration(math.Pow(float64(attemptNo+3), 1.2)) * time.Second
			fut := actor.NewFuture(timeout)
			target := c.Group.GetRandomSyncer()
			actor.EmptyRootContext.RequestWithCustomSender(target, &services.GetTipRequest{
				ChainId: c.TreeDID,
			}, fut.PID())
			res, err = fut.Result()
			attemptNo++
			return err
		},
		retry.Attempts(4),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		return nil, fmt.Errorf("error getting tip: %v", err)
	}
	if res.(*signatures.TreeState).Signature != nil {
		// cache the result to the LRU so future requests to height will
		// return the answer by sending the answer to the subscriber
		actor.EmptyRootContext.Send(c.subscriber, res)
	}
	return res.(*signatures.TreeState), nil
}

// Subscribe returns a future that will return when the height the transaction
// is targeting is complete or an error with the transaction occurs.
//
// TODO: return Subscription instead of actor.Future
func (c *Client) Subscribe(trans *services.AddBlockRequest, timeout time.Duration) *actor.Future {
	if !c.alreadyListening() {
		c.Listen()
	}
	actorContext := actor.EmptyRootContext
	transID := consensus.RequestID(trans)

	fut := actor.NewFuture(timeout)
	killer := actor.NewFuture(timeout + 100*time.Millisecond)
	fut.PipeTo(killer.PID())

	sub := c.stream.Subscribe(func(msgInter interface{}) {
		switch msg := msgInter.(type) {
		case *signatures.TreeState:
			// if the tips are equal then we got a great response and we can go on our merry way
			if bytes.Equal(msg.NewTip, trans.NewTip) {
				actorContext.Send(fut.PID(), msg)
				return
			}

			// if we didn't get an equal tip, but it was at the same height, it means someone else got to us first.
			if msg.Height == trans.Height {
				actorContext.Send(fut.PID(), fmt.Errorf("error signature at same height did not match transaction new tip. Expected %s, got %s", trans.NewTip, msg.NewTip))
				return
			}

			// if the height of the return was greater than this transaction than don't freak out because messages can come in out of order, but
			// log it as an error still because we'd like to minimize these things. Also, don't send a positive, just don't send anything and
			// let timeout handle this if it's actually an error.
			if msg.Height > trans.Height {
				c.log.Error("error received height %d before the height %s was looking for (%d)", msg.Height, transID, trans.Height)
				return
			}
		}
	})

	go func() {
		err := killer.Wait()
		if err != nil {
			c.log.Errorw("error waiting", "err", err)
		}
		c.stream.Unsubscribe(sub)
	}()

	if val, ok := c.cache.Get(string(consensus.RequestID(trans))); ok {
		actorContext.Send(fut.PID(), val)
		return fut
	}

	if val, ok := c.cache.Get(strconv.FormatUint(trans.Height, 10)); ok {
		actorContext.Send(fut.PID(), val)
		return fut
	}

	return fut
}

// SubscribeAll accepts a callback that forwards all CurrentState messages
// broadcasted on tupelo-commits
func (c *Client) SubscribeAll(fn func(msg *signatures.TreeState)) (*Subscription, error) {
	if !c.alreadyListening() {
		c.Listen()
	}

	sub := c.stream.Subscribe(func(msgInter interface{}) {
		switch msg := msgInter.(type) {
		case *signatures.TreeState:
			fn(msg)
		}
	})

	return &Subscription{subscription: sub}, nil
}

// Unsubscribe removes subscription from eventstream
// currently only used for SubscribeAll
func (c *Client) Unsubscribe(s *Subscription) {
	c.stream.Unsubscribe(s.subscription)
}

// SendTransaction sends a transaction to a signer.
func (c *Client) SendTransaction(trans *services.AddBlockRequest) error {
	topic := c.Group.Config().TransactionTopic
	c.log.Debugw("broadcasting transaction", "topic", topic)
	return c.pubsub.Broadcast(topic, trans)
}

func (c *Client) attemptPlayTransactions(tree *consensus.SignedChainTree, treeKey *ecdsa.PrivateKey, remoteTip *cid.Cid, transactions []*transactions.Transaction) (*consensus.AddBlockResponse, error) {
	ctx := context.TODO()
	sw := safewrap.SafeWrap{}

	if remoteTip != nil && cid.Undef.Equals(*remoteTip) {
		remoteTip = nil
	}

	root, err := getRoot(tree)
	if err != nil {
		return nil, fmt.Errorf("error getting root: %v", err)
	}

	var height uint64

	if tree.IsGenesis() {
		height = 0
	} else {
		height = root.Height + 1
	}

	unsignedBlock := &chaintree.BlockWithHeaders{
		Block: chaintree.Block{
			Height:       height,
			PreviousTip:  remoteTip,
			Transactions: transactions,
		},
	}

	blockWithHeaders, err := consensus.SignBlock(unsignedBlock, treeKey)
	if err != nil {
		return nil, fmt.Errorf("error signing: %v", err)
	}

	nodes, err := nodesForTransaction(tree)
	if err != nil {
		return nil, fmt.Errorf("error generating nodes for transaction %v", err)
	}

	storedTip := tree.Tip()

	newChainTree, valid, err := tree.ChainTree.ProcessBlockImmutable(ctx, blockWithHeaders)
	if !valid || err != nil {
		return nil, fmt.Errorf("error processing block (valid: %t): %v", valid, err)
	}

	expectedTip := newChainTree.Dag.Tip
	chainId, err := tree.Id()
	if err != nil {
		return nil, err
	}

	transaction := services.AddBlockRequest{
		PreviousTip: storedTip.Bytes(),
		Height:      blockWithHeaders.Height,
		Payload:     sw.WrapObject(blockWithHeaders).RawData(),
		NewTip:      expectedTip.Bytes(),
		ObjectId:    []byte(chainId),
		State:       nodesToBytes(nodes),
	}

	fut := c.Subscribe(&transaction, 10*time.Second)

	time.Sleep(100 * time.Millisecond)
	c.log.Debugw("sending transaction", "height", transaction.Height, "chainTreeId", chainId)
	err = c.SendTransaction(&transaction)
	if err != nil {
		return nil, fmt.Errorf("error sending transaction %v", err)
	}

	c.log.Debugw("waiting on transaction to complete", "height", transaction.Height,
		"chainTreeId", chainId)

	uncastResp, err := fut.Result()
	if err != nil {
		if err == actor.ErrTimeout {
			c.log.Debugw("transaction failed due to timeout", "error", err, "height", transaction.Height,
				"chainTreeId", chainId)
			return nil, fmt.Errorf(ErrorTimeout)
		}
		c.log.Debugw("transaction failed", "error", err, "height", transaction.Height,
			"chainTreeId", chainId)
		return nil, fmt.Errorf("error response: %v", err)
	}
	if uncastResp == nil {
		c.log.Debugw("transaction timed out", "height", transaction.Height, "chainTreeId", chainId)
		return nil, fmt.Errorf(ErrorTimeout)
	}

	c.log.Debugw("transaction completed successfully", "height", transaction.Height,
		"chainTreeId", chainId)

	var resp *signatures.TreeState
	switch respVal := uncastResp.(type) {
	case *signatures.TreeState:
		resp = respVal
	case error:
		return nil, respVal
	default:
		c.log.Debugw("transaction resulted in an unrecognized response type", "response", respVal)
		return nil, fmt.Errorf("error unrecognized response type: %T", respVal)
	}

	if !bytes.Equal(resp.NewTip, expectedTip.Bytes()) {
		respCid, _ := cid.Cast(resp.NewTip)
		return nil, fmt.Errorf("error, tree updated to different tip - expected: %v - received: %v",
			expectedTip.String(), respCid.String())
	}

	tree.ChainTree = newChainTree

	if tree.Signatures == nil {
		tree.Signatures = make(consensus.SignatureMap)
	}

	tree.Signatures[c.Group.ID] = resp.Signature

	newCid, err := cid.Cast(resp.NewTip)
	if err != nil {
		return nil, fmt.Errorf("error new tip is not parsable CID %v", string(resp.NewTip))
	}

	addResponse := &consensus.AddBlockResponse{
		ChainId:   tree.MustId(),
		Tip:       &newCid,
		Signature: *tree.Signatures[c.Group.ID],
	}

	c.log.Debugw("successfully played transactions")
	return addResponse, nil
}

// PlayTransactions plays transactions in chain tree.
// It retries on timeouts so most of the logic in here is for retries and the meat of the
// transaction-playing code is in attemptPlayTransactions.
func (c *Client) PlayTransactions(tree *consensus.SignedChainTree, treeKey *ecdsa.PrivateKey, remoteTip *cid.Cid, transactions []*transactions.Transaction) (*consensus.AddBlockResponse, error) {
	ctx := context.TODO()

	var (
		resp *consensus.AddBlockResponse
		err  error
	)

	latestRemoteTip := remoteTip

	c.log.Debugw("playing transactions against tree", "numTransactions", len(transactions),
		"maxAttempts", MaxPlayTransactionsAttempts)

	attemptNo := 0
	err = retry.Do(
		func() error {
			attemptNo++
			c.log.Debugw("attempt to play transactions", "attemptNo", attemptNo+1)
			resp, err = c.attemptPlayTransactions(tree, treeKey, latestRemoteTip, transactions)
			signerId := ""
			if resp != nil {
				signerId = resp.SignerId
			}
			c.log.Debugw("attempt ended", "error", err, "response.signerId", signerId)
			return err
		},
		retry.Attempts(MaxPlayTransactionsAttempts),
		retry.RetryIf(func(err error) bool {
			shouldRetry := err.Error() == ErrorTimeout
			c.log.Debugf("should retry playing transactions: %v (%q == %q)", shouldRetry, err.Error(),
				ErrorTimeout)
			return shouldRetry
		}),
		retry.OnRetry(func(n uint, err error) {
			c.log.Debugf("PlayTransactions attempt #%d error: %s", n, err)

			if n > 1 {
				c.log.Debugw("trying to update the tip")
				// Try updating tip in case it has moved forward since the first attempt
				// (possibly due to our transactions succeeding but we just didn't get the response).
				cs, err := c.TipRequest()
				if err != nil {
					return
				}

				if cs.Signature != nil {
					tip, err := cid.Cast(cs.NewTip)
					if err != nil {
						c.log.Errorf("unable to cast remote tip to CID: %v", err)
						return
					}

					if !tip.Equals(tree.Tip()) {
						c.log.Debugw("tip is out of date, updating")
						newTipNode, err := tree.ChainTree.Dag.Get(ctx, tip)
						if err != nil {
							c.log.Errorf("error getting new tip node from DAG: %v", err)
							return
						}

						if newTipNode == nil {
							c.log.Errorw("latest tip node is not present", "cid", tip.String())
							return
						} else {
							latestRemoteTip = &tip
							tree.ChainTree.Dag.Tip = tip
						}
					}
				}
			}
		}),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		c.log.Debugw("PlayTransactions failed", "error", err)
		return nil, err
	}

	c.log.Debugw("PlayTransactions succeeded")
	return resp, nil
}

func (c *Client) TokenPayloadForTransaction(chain *chaintree.ChainTree, tokenName *consensus.TokenName, sendTokenTxId string, sendTxState *signatures.TreeState) (*transactions.TokenPayload, error) {
	return consensus.TokenPayloadForTransaction(chain, tokenName, sendTokenTxId, sendTxState)
}

func getRoot(sct *consensus.SignedChainTree) (*chaintree.RootNode, error) {
	ctx := context.TODO()
	ct := sct.ChainTree
	unmarshaledRoot, err := ct.Dag.Get(ctx, ct.Dag.Tip)
	if unmarshaledRoot == nil || err != nil {
		return nil, fmt.Errorf("error,missing root: %v", err)
	}

	root := &chaintree.RootNode{}

	err = cbornode.DecodeInto(unmarshaledRoot.RawData(), root)
	if err != nil {
		return nil, fmt.Errorf("error decoding root: %v", err)
	}
	return root, nil
}

// This method should calculate all necessary nodes that need to be sent for verification.
// Currently this takes
// - the entire resolved tree/ of the existing tree
// - the nodes for chain/end, but not resolving through the previous tip
func nodesForTransaction(existingSignedTree *consensus.SignedChainTree) ([]format.Node, error) {
	ctx := context.TODO()
	existingTree := existingSignedTree.ChainTree

	treeNodes, err := existingTree.Dag.NodesForPathWithDecendants(ctx, []string{"tree"})
	if err != nil {
		return nil, fmt.Errorf("error getting tree nodes: %v", err)
	}

	// Validation needs all the nodes for chain/end, but not past chain/end. aka no need to
	// resolve the end node, since that would fetch all the nodes of from the previous tip.
	// Also, on genesis state chain/end is nil, so deal with that
	var chainNodes []format.Node
	if existingSignedTree.IsGenesis() {
		chainNodes, err = existingTree.Dag.NodesForPath(ctx, []string{chaintree.ChainLabel})
	} else {
		chainNodes, err = existingTree.Dag.NodesForPath(ctx, []string{chaintree.ChainLabel, chaintree.ChainEndLabel})
	}
	if err != nil {
		return nil, fmt.Errorf("error getting chain nodes: %v", err)
	}

	// subtract 1 to only include tip node once
	nodes := make([]format.Node, len(treeNodes)+len(chainNodes)-1)
	i := 0
	for _, node := range treeNodes {
		nodes[i] = node
		i++
	}
	for _, node := range chainNodes {
		if node.Cid().Equals(existingTree.Dag.Tip) {
			continue
		}
		nodes[i] = node
		i++
	}

	return nodes, nil
}

func nodesToBytes(nodes []format.Node) [][]byte {
	returnBytes := make([][]byte, len(nodes))
	for i, n := range nodes {
		returnBytes[i] = n.RawData()
	}
	return returnBytes
}
