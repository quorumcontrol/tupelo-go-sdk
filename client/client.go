package client

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/safewrap"
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

// Client represents a Tupelo client.
type Client struct {
	Group  *types.NotaryGroup
	log    *zap.SugaredLogger
	pubsub remote.PubSub
}

// New instantiates a Client for a notary group.
func New(group *types.NotaryGroup, pubsub remote.PubSub) *Client {
	return &Client{
		Group:  group,
		log:    middleware.Log.Named("client"),
		pubsub: pubsub,
	}
}

// Stop stops a Client.
func (c *Client) Stop() {
	// no op
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
func (c *Client) Subscribe(treeDid string, expectedTip cid.Cid, timeout time.Duration) *actor.Future {
	fut := actor.NewFuture(timeout)
	act := c.pubsub.Subscribe(actor.EmptyRootContext, treeDid, fut.PID())

	// the future we'll use to kill the subscriber actor
	killFut := actor.NewFuture(timeout + 1*time.Second)
	fut.PipeTo(killFut.PID())
	go func() {
		err := killFut.Wait()
		if err != nil {
			c.log.Debugw("future timed out in client subscribe", "did", treeDid, "tip", expectedTip)
		}
		act.Stop()
	}()
	return fut
}

// SendTransaction sends a transaction to a signer.
func (c *Client) SendTransaction(trans *messages.Transaction) error {
	return c.pubsub.Broadcast(TransactionBroadcastTopic, trans)
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

	fut := c.Subscribe(tree.MustId(), expectedTip, 60*time.Second)

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
