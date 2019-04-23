package client

import (
	"testing"
	"time"

	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/testhelpers"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/types"
	"github.com/stretchr/testify/require"
)

func TestSubscription(t *testing.T) {
	middleware.SetLogLevel("debug")
	ng := types.NewNotaryGroup("testSubscriptions")

	trans := testhelpers.NewValidTransaction(t)

	pubSubSystem := remote.NewSimulatedPubSub()

	t.Run("works with current state", func(t *testing.T) {
		client := New(ng, string(trans.ObjectID), pubSubSystem)
		defer client.Stop()

		fut := client.Subscribe(&trans, 1*time.Second)
		time.Sleep(100 * time.Millisecond)
		currState := &messages.CurrentState{
			Signature: &messages.Signature{
				ObjectID: trans.ObjectID,
				Height:   trans.Height,
			},
		}
		pubSubSystem.Broadcast(string(trans.ObjectID), currState)

		res, err := fut.Result()
		require.Nil(t, err)
		require.Equal(t, currState, res)
	})

	t.Run("works with errors", func(t *testing.T) {
		client := New(ng, string(trans.ObjectID), pubSubSystem)
		defer client.Stop()

		fut := client.Subscribe(&trans, 1*time.Second)
		time.Sleep(100 * time.Millisecond)
		msgErr := &messages.Error{
			Source: string(trans.ID()),
		}
		pubSubSystem.Broadcast(string(trans.ObjectID), msgErr)

		res, err := fut.Result()
		require.Nil(t, err)
		require.Equal(t, msgErr, res)
	})

}
