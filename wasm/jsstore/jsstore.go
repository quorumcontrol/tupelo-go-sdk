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

// JSStore is a go wrapper to use a javascript nodestore from wasm
// it implements the nodestore.DagStore interface after being passed an underlying
// javascript block-service implementation (for instance: https://github.com/ipfs/js-ipfs-block-service )
type JSStore struct {
	nodestore.DagStore
	bridged js.Value
}

// New expects a javascript ipfs-block-service (javascript flavor)
// and wraps it to make it compatible with the nodestore.DagStore interface (which is currently just format.DagService)
func New(bridged js.Value) nodestore.DagStore {
	return &JSStore{
		bridged: bridged,
	}
}

func (jss *JSStore) Remove(ctx context.Context, c cid.Cid) error {
	respCh := make(chan error)
	go func() {
		promise := jss.bridged.Call("delete", helpers.CidToJSCID(c))
		onSuccess := js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
			respCh <- nil
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
		return resp
	case <-ctx.Done():
		return fmt.Errorf("context done")
	}
	return fmt.Errorf("should never get here")
}

func (jss *JSStore) RemoveMany(ctx context.Context, cids []cid.Cid) error {
	// for now do the slow thing and just loop over, javascript has a putMany as well, but more annoying to serialize
	for _, n := range cids {
		err := jss.Remove(ctx, n)
		if err != nil {
			return fmt.Errorf("error adding: %v", err)
		}
	}
	return nil
}

func (jss *JSStore) AddMany(ctx context.Context, nodes []format.Node) error {
	// for now do the slow thing and just loop over, javascript has a putMany as well, but more annoying to serialize
	for _, n := range nodes {
		err := jss.Add(ctx, n)
		if err != nil {
			return fmt.Errorf("error adding: %v", err)
		}
	}
	return nil
}

func (jss *JSStore) GetMany(ctx context.Context, cids []cid.Cid) <-chan *format.NodeOption {
	panic("don't CALL ME, I'm unimplemented and full of choclate")
}

func (jss *JSStore) Add(ctx context.Context, n format.Node) error {
	respCh := make(chan error)
	go func() {
		promise := jss.bridged.Call("put", nodeToJSBlock(n))
		onSuccess := js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
			respCh <- nil
			return nil
		})
		onError := js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
			err := fmt.Errorf("error from js: %s", args[0].String())
			go fmt.Println("err: ", err.Error())
			respCh <- err
			return nil
		})
		promise.Call("then", onSuccess, onError)
	}()

	select {
	case resp := <-respCh:
		return resp
	case <-ctx.Done():
		return fmt.Errorf("context done")
	}
	return fmt.Errorf("should never get here")
}

func (jss *JSStore) Get(ctx context.Context, c cid.Cid) (format.Node, error) {
	respCh := make(chan interface{})
	go func() {
		sw := safewrap.SafeWrap{}
		promise := jss.bridged.Call("get", helpers.CidToJSCID(c))
		onSuccess := js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
			jsBlock := args[0]
			bits := helpers.JsBufferToBytes(jsBlock.Get("data"))
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

func nodeToJSBlock(n format.Node) js.Value {
	jsBlock := js.Global().Call("require", "ipfs-block")
	data := helpers.SliceToJSBuffer(n.RawData())
	return jsBlock.New(data, helpers.CidToJSCID(n.Cid()))
}
