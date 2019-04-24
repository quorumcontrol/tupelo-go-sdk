package client

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/eventstream"
	lru "github.com/hashicorp/golang-lru"
	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/transactions"
	"go.uber.org/zap"

	"github.com/quorumcontrol/tupelo-go-client/consensus"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/types"
)

// TransactionBroadcastTopic is the topic from which clients
// broadcast their transactions to the NotaryGroup
const TransactionBroadcastTopic = "tupelo-transaction-broadcast"

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

// New instantiates a Client specific to a ChainTree/NotaryGroup
func New(group *types.NotaryGroup, treeDid string, pubsub remote.PubSub) *Client {
	cache, err := lru.New(10000)
	if err != nil {
		panic(fmt.Errorf("error generating LRU: %v", err))
	}
	return &Client{
		Group:   group,
		TreeDID: treeDid,
		log:     middleware.Log.Named("client"),
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
	if !c.alreadyListening() {
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
	case *messages.CurrentState:
		heightString := strconv.FormatUint(msg.Signature.Height, 10)
		existed, _ := c.cache.ContainsOrAdd(heightString, msg)
		if !existed {
			c.log.Debugw("publishing current state", "height", heightString)
			c.stream.Publish(msg)
		}
	case *messages.Error:
		existed, _ := c.cache.ContainsOrAdd(string(msg.Source), msg)
		if !existed {
			c.log.Debugw("publishing error", "tx", string(msg.Source))
			c.stream.Publish(msg)
		}
	default:
		c.log.Debugw("unknown message received", "type", reflect.TypeOf(msg).String())
	}
}

// TipRequest requests the tip of a chain tree.
func (c *Client) TipRequest() (*messages.CurrentState, error) {
	target := c.Group.GetRandomSyncer()
	fut := actor.NewFuture(10 * time.Second)
	actor.EmptyRootContext.RequestWithCustomSender(target, &messages.GetTip{
		ObjectID: []byte(c.TreeDID),
	}, fut.PID())
	res, err := fut.Result()
	if err != nil {
		return nil, fmt.Errorf("error getting tip: %v", err)
	}
	// cache the result to the LRU so future requests to height will
	// return the answer by sending the answer to the subscriber
	actor.EmptyRootContext.Send(c.subscriber, res)
	return res.(*messages.CurrentState), nil
}

// Subscribe returns a future that will return when the height the transaction
// is targeting is complete or an error with the transaction occurs.
func (c *Client) Subscribe(trans *messages.Transaction, timeout time.Duration) *actor.Future {
	if !c.alreadyListening() {
		c.Listen()
	}
	actorContext := actor.EmptyRootContext
	transID := trans.ID()

	fut := actor.NewFuture(timeout)
	killer := actor.NewFuture(timeout + 100*time.Millisecond)
	fut.PipeTo(killer.PID())

	sub := c.stream.Subscribe(func(msg interface{}) {
		actorContext.Send(fut.PID(), msg)
	})
	sub.WithPredicate(func(msgInterface interface{}) bool {
		switch msg := msgInterface.(type) {
		case *messages.CurrentState:
			if msg.Signature.Height == trans.Height {
				return true
			}
		case *messages.Error:
			if msg.Source == string(transID) {
				return true
			}
		}
		return false
	})

	go func() {
		err := killer.Wait()
		if err != nil {
			c.log.Errorw("error waiting", "err", err)
		}
		c.stream.Unsubscribe(sub)
	}()

	if val, ok := c.cache.Get(string(trans.ID())); ok {
		actorContext.Send(fut.PID(), val)
		return fut
	}

	if val, ok := c.cache.Get(strconv.FormatUint(trans.Height, 10)); ok {
		actorContext.Send(fut.PID(), val)
		return fut
	}

	return fut
}

// SendTransaction sends a transaction to a signer.
func (c *Client) SendTransaction(trans *messages.Transaction) error {
	return c.pubsub.Broadcast(TransactionBroadcastTopic, trans)
}

// PlayTransactions plays transactions in chain tree.
func (c *Client) PlayTransactions(tree *consensus.SignedChainTree, treeKey *ecdsa.PrivateKey, remoteTip *cid.Cid, transactions []*transactions.Transaction) (*consensus.AddBlockResponse, error) {
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

	//TODO: only send the necessary nodes
	cborNodes, err := tree.ChainTree.Dag.Nodes()
	if err != nil {
		return nil, fmt.Errorf("error getting nodes: %v", err)
	}
	nodes := make([][]byte, len(cborNodes))
	for i, node := range cborNodes {
		nodes[i] = node.RawData()
	}
	storedTip := tree.Tip()

	newTree, err := chaintree.NewChainTree(tree.ChainTree.Dag, tree.ChainTree.BlockValidators, tree.ChainTree.Transactors)
	if err != nil {
		return nil, fmt.Errorf("error creating new tree: %v", err)
	}
	valid, err := newTree.ProcessBlock(blockWithHeaders)
	if !valid || err != nil {
		return nil, fmt.Errorf("error processing block (valid: %t): %v", valid, err)
	}

	expectedTip := newTree.Dag.Tip

	transaction := messages.Transaction{
		PreviousTip: storedTip.Bytes(),
		Height:      blockWithHeaders.Height,
		Payload:     sw.WrapObject(blockWithHeaders).RawData(),
		NewTip:      expectedTip.Bytes(),
		ObjectID:    []byte(tree.MustId()),
		State:       nodes,
	}

	fut := c.Subscribe(&transaction, 60*time.Second)

	time.Sleep(100 * time.Millisecond)
	err = c.SendTransaction(&transaction)
	if err != nil {
		panic(fmt.Errorf("error sending transaction %v", err))
	}

	uncastResp, err := fut.Result()
	if err != nil {
		return nil, fmt.Errorf("error response: %v", err)
	}

	if uncastResp == nil {
		return nil, fmt.Errorf("error timeout")
	}

	var resp *messages.CurrentState
	switch respVal := uncastResp.(type) {
	case *messages.Error:
		return nil, fmt.Errorf("error response: %v", respVal)
	case *messages.CurrentState:
		resp = respVal
	default:
		return nil, fmt.Errorf("error unrecognized response type: %T", respVal)
	}

	if !bytes.Equal(resp.Signature.NewTip, expectedTip.Bytes()) {
		respCid, _ := cid.Cast(resp.Signature.NewTip)
		return nil, fmt.Errorf("error, tree updated to different tip - expected: %v - received: %v", expectedTip.String(), respCid.String())
	}

	success, err := tree.ChainTree.ProcessBlock(blockWithHeaders)
	if !success || err != nil {
		return nil, fmt.Errorf("error processing block (valid: %t): %v", valid, err)
	}

	tree.Signatures[c.Group.ID] = *resp.Signature

	newCid, err := cid.Cast(resp.Signature.NewTip)
	if err != nil {
		return nil, fmt.Errorf("error new tip is not parsable CID %v", string(resp.Signature.NewTip))
	}

	addResponse := &consensus.AddBlockResponse{
		ChainId:   tree.MustId(),
		Tip:       &newCid,
		Signature: tree.Signatures[c.Group.ID],
	}

	if tree.Signatures == nil {
		tree.Signatures = make(consensus.SignatureMap)
	}

	return addResponse, nil
}

func getRoot(sct *consensus.SignedChainTree) (*chaintree.RootNode, error) {
	ct := sct.ChainTree
	unmarshaledRoot, err := ct.Dag.Get(ct.Dag.Tip)
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
