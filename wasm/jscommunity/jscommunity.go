// +build wasm

package jscommunity

import (
	"context"
	"syscall/js"

	cbornode "github.com/ipfs/go-ipld-cbor"

	"github.com/ipfs/go-datastore"

	"github.com/gogo/protobuf/proto"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jsstore"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/helpers"

	hamt "github.com/quorumcontrol/go-hamt-ipld"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/then"
)

func init() {
	typecaster.AddType(signatures.CurrentState{})
	cbornode.RegisterCborType(signatures.CurrentState{})
}

func GetCurrentState(ctx context.Context, jsCid js.Value, jsBlockService js.Value, jsDid js.Value) *then.Then {
	t := then.New()
	go func() {
		tip, err := helpers.JsCidToCid(jsCid)
		if err != nil {
			t.Reject(errors.Wrap(err, "error converting CID").Error())
			return
		}
		store := jsstore.New(jsBlockService)
		did := jsDid.String()

		hamtstore := hamt.CSTFromDAG(store)
		n, err := hamt.LoadNode(ctx, hamtstore, tip)
		if err != nil {
			t.Reject(errors.Wrap(err, "error getting hamt node").Error())
			return
		}
		key := datastore.KeyWithNamespaces([]string{"states", did + ":current"})
		storedMapState, err := n.Find(ctx, key.String())
		if err != nil {
			t.Reject(err.Error())
			return
		}
		currState, err := hamtReturnToCurrentState(storedMapState)
		if err != nil {
			t.Reject(errors.Wrap(err, "error converting to current state").Error())
			return
		}
		bits, err := proto.Marshal(currState)
		if err != nil {
			t.Reject(errors.Wrap(err, "error marshaling").Error())
			return
		}
		t.Resolve(helpers.SliceToJSArray(bits))
	}()
	return t
}

func hamtReturnToCurrentState(storedMap interface{}) (*signatures.CurrentState, error) {
	var cs signatures.CurrentState
	if err := typecaster.ToType(storedMap, &cs); err != nil {
		return nil, errors.Wrap(err, "error typecasting")
	}
	return &cs, nil
}
