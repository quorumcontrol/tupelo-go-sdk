// +build wasm

package jscommunity

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/gogo/protobuf/proto"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jsstore"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/helpers"

	hamt "github.com/quorumcontrol/go-hamt-ipld"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/then"
)

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
		storedMapState, err := n.Find(ctx, fmt.Sprintf("current/%s:current", did))
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
		t.Resolve(bits)
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
