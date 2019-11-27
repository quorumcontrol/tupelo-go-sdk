// +build wasm

package jscommunity

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"syscall/js"

	"github.com/ethereum/go-ethereum/crypto"
	ptypes "github.com/gogo/protobuf/types"
	pb "github.com/quorumcontrol/messages/v2/build/go/community"

	"github.com/ipfs/go-datastore"

	"github.com/gogo/protobuf/proto"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jsstore"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/helpers"

	hamt "github.com/quorumcontrol/go-hamt-ipld"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/then"
)

func HashToShardNumber(topicName string, shardCount int) int {
	hsh := sha256.Sum256([]byte(topicName))
	shardNum, _ := binary.Uvarint(hsh[:])
	return int(shardNum % uint64(shardCount))
}

func GetSendableBytes(envelopeBits []byte, keyBits []byte) *then.Then {
	t := then.New()
	go func() {
		key, err := crypto.ToECDSA(keyBits)
		if err != nil {
			t.Reject(errors.Wrap(err, "error converting bytes to key").Error())
			return
		}

		env := new(pb.Envelope)
		err = env.Unmarshal(envelopeBits)
		if err != nil {
			t.Reject(errors.Wrap(err, "error unmarshaling envelope").Error())
			return
		}

		env.Sign(key)

		any, err := ptypes.MarshalAny(env)
		if err != nil {
			t.Reject(errors.Wrap(err, "error turning into any").Error())
			return
		}

		marshaled, err := any.Marshal()
		if err != nil {
			t.Reject(errors.Wrap(err, "error marshaling").Error())
			return
		}
		t.Resolve(helpers.SliceToJSBuffer(marshaled))
	}()

	return t
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

func hamtReturnToCurrentState(storedMap interface{}) (*signatures.TreeState, error) {
	var cs signatures.TreeState
	if err := typecaster.ToType(storedMap, &cs); err != nil {
		return nil, errors.Wrap(err, "error typecasting")
	}
	return &cs, nil
}
