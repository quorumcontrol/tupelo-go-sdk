package client

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	"github.com/quorumcontrol/chaintree/nodestore"
)

type cidTracker map[cid.Cid]struct{}

/**
storeWrapper keeps tracks of all the Gets and Adds in a DagStore.
We use this because we want to know what from the previous state needs to be sent up to a Tupelo Signer
So when we're playing transactions against a tree, we swap out its store for this one and keep track of all the Gets and Adds.
The reason for keeping track of Adds is that a second Tx in a block might reference some *new* nodes created by the first,
but there is no reason to send those up to Tupelo in the state of the Tx.
*/
type storeWrapper struct {
	nodestore.DagStore
	touched  cidTracker
	newNodes cidTracker
}

func (sw *storeWrapper) Get(ctx context.Context, id cid.Cid) (format.Node, error) {
	n, err := sw.DagStore.Get(ctx, id)
	if err == nil && n != nil {
		if _, ok := sw.newNodes[id]; !ok {
			sw.touched[id] = struct{}{}
		}
	}
	return n, err
}

func (sw *storeWrapper) Add(ctx context.Context, n format.Node) error {
	err := sw.DagStore.Add(ctx, n)
	if err == nil {
		sw.newNodes[n.Cid()] = struct{}{}
	}
	return err
}

func (sw *storeWrapper) touchedCids() []cid.Cid {
	ids := make([]cid.Cid, len(sw.touched))
	i := 0
	for k := range sw.touched {
		ids[i] = k
		i++
	}
	return ids
}

func (sw *storeWrapper) touchedNodes(ctx context.Context) ([]format.Node, error) {
	ids := sw.touchedCids()
	nodes := make([]format.Node, len(ids))
	for i, id := range ids {
		n, err := sw.DagStore.Get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("error getting node: %v", err)
		}
		nodes[i] = n
	}
	return nodes, nil
}

func wrapStoreForRefCounting(store nodestore.DagStore) *storeWrapper {
	return &storeWrapper{
		DagStore: store,
		touched:  make(cidTracker),
		newNodes: make(cidTracker),
	}
}
