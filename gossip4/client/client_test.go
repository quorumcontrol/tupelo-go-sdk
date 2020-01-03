package client

import (
	"errors"
	"testing"
	"time"

	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip4/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip4/testhelpers"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip4/types"
	"github.com/stretchr/testify/require"
)

func TestSubscription(t *testing.T) {
	ng := types.NewNotaryGroup("testSubscriptions")

	trans := testhelpers.NewValidTransaction(t)

	pubSubSystem := remote.NewSimulatedPubSub()

	t.Run("works with current state", func(t *testing.T) {
		client := New(ng, string(trans.ObjectId), pubSubSystem)
		defer client.Stop()

		fut := client.Subscribe(&trans, 1*time.Second)
		time.Sleep(100 * time.Millisecond)
		currState := &signatures.TreeState{
			ObjectId:  trans.ObjectId,
			Height:    trans.Height,
			NewTip:    trans.NewTip,
			Signature: &signatures.Signature{},
		}
		err := pubSubSystem.Broadcast(string(trans.ObjectId), currState)
		require.Nil(t, err)

		res, err := fut.Result()
		require.Nil(t, err)
		require.Equal(t, currState, res)
	})

	t.Run("errors when height has non-matching tip", func(t *testing.T) {
		client := New(ng, string(trans.ObjectId), pubSubSystem)
		defer client.Stop()

		fut := client.Subscribe(&trans, 1*time.Second)
		time.Sleep(100 * time.Millisecond)
		currState := &signatures.TreeState{
			ObjectId:  trans.ObjectId,
			Height:    trans.Height,
			NewTip:    []byte("somebadtip"),
			Signature: &signatures.Signature{},
		}
		err := pubSubSystem.Broadcast(string(trans.ObjectId), currState)
		require.Nil(t, err)

		res, err := fut.Result()
		require.Nil(t, err)
		require.IsType(t, errors.New("error"), res)
	})
}

func TestSubscribeAll(t *testing.T) {
	ng := types.NewNotaryGroup("testSubscriptions")
	trans := testhelpers.NewValidTransaction(t)
	pubSubSystem := remote.NewSimulatedPubSub()
	did := string(trans.ObjectId)

	client := New(ng, did, pubSubSystem)
	defer client.Stop()

	states := make([]*signatures.TreeState, 2)
	i := 0

	sub, err := client.SubscribeAll(func(msg *signatures.TreeState) {
		states[i] = msg
		i++
	})
	defer client.Unsubscribe(sub)
	require.Nil(t, err)

	time.Sleep(100 * time.Millisecond)
	state1 := &signatures.TreeState{
		ObjectId:  trans.ObjectId,
		Height:    trans.Height,
		NewTip:    trans.NewTip,
		Signature: &signatures.Signature{},
	}
	err = pubSubSystem.Broadcast(did, state1)
	require.Nil(t, err)

	state2 := &signatures.TreeState{
		ObjectId:  trans.ObjectId,
		Height:    trans.Height + 1,
		NewTip:    trans.NewTip,
		Signature: &signatures.Signature{},
	}
	err = pubSubSystem.Broadcast(did, state2)
	require.Nil(t, err)

	time.Sleep(100 * time.Millisecond)
	require.Equal(t, len(states), 2)
	require.Equal(t, states[0], state1)
	require.Equal(t, states[1], state2)
}
