package client

import (
	"github.com/quorumcontrol/messages/build/go/signatures"
	"errors"
	"testing"
	"time"

	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/testhelpers"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
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
		currState := &signatures.CurrentState{
			Signature: &signatures.Signature{
				ObjectId: trans.ObjectId,
				Height:   trans.Height,
				NewTip:   trans.NewTip,
			},
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
		currState := &signatures.CurrentState{
			Signature: &signatures.Signature{
				ObjectId: trans.ObjectId,
				Height:   trans.Height,
				NewTip:   []byte("somebadtip"),
			},
		}
		err := pubSubSystem.Broadcast(string(trans.ObjectId), currState)
		require.Nil(t, err)

		res, err := fut.Result()
		require.Nil(t, err)
		require.IsType(t, errors.New("error"), res)
	})
}
