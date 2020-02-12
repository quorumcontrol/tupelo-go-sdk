package client

import (
	"context"
	"fmt"

	format "github.com/ipfs/go-ipld-format"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/v2/build/go/services"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip/blocks"
)

// NewAddBlockRequest creates an add block request for sending in the client
func (c *Client) NewAddBlockRequest(ctx context.Context, tree *consensus.SignedChainTree, opts ...blocks.Option) (*services.AddBlockRequest, error) {
	blockWithHeaders, err := blocks.NewBlockWithHeaders(ctx, tree, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating block: %w", err)
	}

	if len(tree.ChainTree.BlockValidators) == 0 {
		// we run the block validators to save devs from themselves
		// and catch anything we know will be rejected by the NotaryGroup
		validators, err := c.Group.BlockValidators(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting notary group block validators: %v", err)
		}
		tree.ChainTree.BlockValidators = validators
	}

	trackedTree, tracker, err := refTrackingChainTree(ctx, tree.ChainTree)
	if err != nil {
		return nil, fmt.Errorf("error creating reference tracker: %v", err)
	}

	valid, err := trackedTree.ProcessBlock(ctx, blockWithHeaders)
	if !valid || err != nil {
		return nil, fmt.Errorf("error processing block (valid: %t): %v", valid, err)
	}

	var state [][]byte
	// only need state after the first Tx
	if blockWithHeaders.Height > 0 {
		// Grab the nodes that were actually used:
		nodes, err := tracker.touchedNodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting node: %v", err)
		}
		state = nodesToBytes(nodes)
	}

	expectedTip := trackedTree.Dag.Tip
	chainID, err := tree.Id()
	if err != nil {
		return nil, err
	}

	sw := safewrap.SafeWrap{}
	payload := sw.WrapObject(blockWithHeaders).RawData()

	return &services.AddBlockRequest{
		PreviousTip: tree.Tip().Bytes(),
		Height:      blockWithHeaders.Height,
		Payload:     payload,
		NewTip:      expectedTip.Bytes(),
		ObjectId:    []byte(chainID),
		State:       state,
	}, nil
}

// Creating a new dag & chaintree with a tracked datastore to ensure that only
// the necessary nodes are sent to the signers for a processed block. Processing
// blocks on the tracked chaintree will keep track of which nodes are accessed.
func refTrackingChainTree(ctx context.Context, untrackedTree *chaintree.ChainTree) (*chaintree.ChainTree, *storeWrapper, error) {
	originalStore := untrackedTree.Dag.Store
	tracker := wrapStoreForRefCounting(originalStore)
	trackedTree, err := chaintree.NewChainTree(ctx, dag.NewDag(ctx, untrackedTree.Dag.Tip, tracker), untrackedTree.BlockValidators, untrackedTree.Transactors)

	return trackedTree, tracker, err
}

func nodesToBytes(nodes []format.Node) [][]byte {
	returnBytes := make([][]byte, len(nodes))
	for i, n := range nodes {
		returnBytes[i] = n.RawData()
	}

	return returnBytes
}
