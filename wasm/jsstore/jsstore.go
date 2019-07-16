package jsstore

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/quorumcontrol/chaintree/safewrap"

	"github.com/quorumcontrol/tupelo-go-sdk/wasm/helpers"

	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"

	"github.com/quorumcontrol/chaintree/nodestore"
)

// type DAGService interface {
// 	NodeGetter
// 	NodeAdder

// 	// Remove removes a node from this DAG.
// 	//
// 	// Remove returns no error if the requested node is not present in this DAG.
// 	Remove(context.Context, cid.Cid) error

// 	// RemoveMany removes many nodes from this DAG.
// 	//
// 	// It returns success even if the nodes were not present in the DAG.
// 	RemoveMany(context.Context, []cid.Cid) error
// }

// // The basic Node resolution service.
// type NodeGetter interface {
// 	// Get retrieves nodes by CID. Depending on the NodeGetter
// 	// implementation, this may involve fetching the Node from a remote
// 	// machine; consider setting a deadline in the context.
// 	Get(context.Context, cid.Cid) (Node, error)

// 	// GetMany returns a channel of NodeOptions given a set of CIDs.
// 	GetMany(context.Context, []cid.Cid) <-chan *NodeOption
// }

// // NodeAdder adds nodes to a DAG.
// type NodeAdder interface {
// 	// Add adds a node to this DAG.
// 	Add(context.Context, Node) error

// 	// AddMany adds many nodes to this DAG.
// 	//
// 	// Consider using the Batch NodeAdder (`NewBatch`) if you make
// 	// extensive use of this function.
// 	AddMany(context.Context, []Node) error
// }

// INSTANCE
// setExchange
// unsetExchange
// hasExchange
// put
// putMany
// get
// getMany
// delete

// JSStore is a go wrapper to use a javascript nodestore from wasm
type JSStore struct {
	format.DAGService

	bridged js.Value
}

// New expects a javascript ipfs-block-service (javascript flavor)
// and wraps it to make it compatible with the nodestore.DagStore interface (which is currently just format.DagService)
func New(bridged js.Value) nodestore.DagStore {
	return &JSStore{
		bridged: bridged,
	}
}

func (jss *JSStore) Get(ctx context.Context, c cid.Cid) (format.Node, error) {
	respCh := make(chan interface{})
	go func() {
		sw := safewrap.SafeWrap{}
		promise := jss.bridged.Call("get", cidToJSCID(c))
		onSuccess := js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
			jsBlock := args[0]
			data := jsBlock.Get("data")
			go fmt.Printf("data: %s", data.String())
			bits := helpers.JsBufferToBytes(data)
			n := sw.Decode(bits)
			if sw.Err != nil {
				respCh <- fmt.Errorf("error decoding: %v", sw.Err)
				return nil
			}
			respCh <- n
			return nil
		})
		onError := js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
			err := fmt.Errorf("error from js: %s", args[0].String())
			respCh <- err
			return nil
		})
		promise.Call("then", onSuccess, onError)
	}()

	select {
	case resp := <-respCh:
		switch msg := resp.(type) {
		case error:
			return nil, msg
		case format.Node:
			return msg, nil
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("context done")
	}
	return nil, fmt.Errorf("should never get here")
}

func cidToJSCID(c cid.Cid) js.Value {
	jsCids := js.Global().Call("require", js.ValueOf("cids"))
	bits := c.Bytes()
	jsBits := helpers.SliceToJSBuffer(bits)
	return jsCids.New(jsBits)
}
