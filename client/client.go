package client

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/AsynkronIT/protoactor-go/plugin"
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/safewrap"
	"go.uber.org/zap"

	"github.com/quorumcontrol/tupelo-go-client/consensus"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/types"
)

// Broadcaster is the interface a client needs to send
// transactions to the network
type Broadcaster interface {
	Broadcast(msg messages.WireMessage) error
}

// Client represents a Tupelo client.
type Client struct {
	Group            *types.NotaryGroup
	log              *zap.SugaredLogger
	subscriberActors []*actor.PID
	broadcaster      Broadcaster
}

type subscriberActor struct {
	middleware.LogAwareHolder

	ch      chan interface{}
	timeout time.Duration
}

func newSubscriberActorProps(ch chan interface{}, timeout time.Duration) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor {
		return &subscriberActor{
			ch:      ch,
			timeout: timeout,
		}
	}).WithReceiverMiddleware(
		middleware.LoggingMiddleware,
		plugin.Use(&middleware.LogPlugin{}),
	)
}

func (sa *subscriberActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		ctx.SetReceiveTimeout(sa.timeout)
	case *actor.ReceiveTimeout:
		ctx.Self().Poison()
		sa.ch <- msg
		close(sa.ch)
	case *actor.Terminated:
		// For some reason we never seem to receive this when we timeout and self-poison
		close(sa.ch)
	case *messages.CurrentState:
		sa.ch <- msg
		ctx.Respond(&messages.TipSubscription{
			ObjectID:    msg.Signature.ObjectID,
			TipValue:    msg.Signature.NewTip,
			Unsubscribe: true,
		})
		ctx.Self().Poison()
	case *messages.Error:
		sa.ch <- msg
		ctx.Self().Poison()
	}
}

// New instantiates a Client for a notary group.
func New(group *types.NotaryGroup, broadcaster Broadcaster) *Client {
	return &Client{
		Group:       group,
		log:         middleware.Log.Named("client"),
		broadcaster: broadcaster,
	}
}

// Stop stops a Client.
func (c *Client) Stop() {
	for _, act := range c.subscriberActors {
		act.Stop()
	}
}

// TipRequest requests the tip of a chain tree.
func (c *Client) TipRequest(chainID string) (*messages.CurrentState, error) {
	target := c.Group.GetRandomSyncer()
	fut := actor.NewFuture(10 * time.Second)
	actor.EmptyRootContext.RequestWithCustomSender(target, &messages.GetTip{
		ObjectID: []byte(chainID),
	}, fut.PID())
	res, err := fut.Result()
	if err != nil {
		return nil, fmt.Errorf("error getting tip: %v", err)
	}
	return res.(*messages.CurrentState), nil
}

// Subscribe creates a subscription to a chain tree.
func (c *Client) Subscribe(signer *types.Signer, treeDid string, expectedTip cid.Cid, timeout time.Duration) (chan interface{}, error) {
	ch := make(chan interface{}, 1)
	act := actor.EmptyRootContext.SpawnPrefix(newSubscriberActorProps(ch, timeout), "sub-"+treeDid)
	c.subscriberActors = append(c.subscriberActors, act)
	actor.EmptyRootContext.RequestWithCustomSender(signer.Actor, &messages.TipSubscription{
		ObjectID: []byte(treeDid),
		TipValue: expectedTip.Bytes(),
	}, act)
	return ch, nil
}

// SendTransaction sends a transaction to a signer.
func (c *Client) SendTransaction(trans *messages.Transaction) error {
	return c.broadcaster.Broadcast(trans)
}

// PlayTransactions plays transactions in chain tree.
func (c *Client) PlayTransactions(tree *consensus.SignedChainTree, treeKey *ecdsa.PrivateKey, remoteTip *cid.Cid, transactions []*chaintree.Transaction) (*consensus.AddBlockResponse, error) {
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

	target := c.Group.GetRandomSigner()

	respChan, err := c.Subscribe(target, tree.MustId(), expectedTip, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error subscribing: %v", err)
	}

	err = c.SendTransaction(&transaction)
	if err != nil {
		panic(fmt.Errorf("error sending transaction %v", err))
	}

	uncastResp := <-respChan

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
