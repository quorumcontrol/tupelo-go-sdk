package remote

import (
	"github.com/quorumcontrol/messages/build/go/services"
	"bytes"
	"context"
	"crypto/ecdsa"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/middleware"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/golang/protobuf/proto"
)

func TestPubSub(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	keys := make([]*ecdsa.PrivateKey, 3)
	for i := 0; i < len(keys); i++ {
		ecdsaKey, err := crypto.GenerateKey()
		if err != nil {
			t.Fatalf("error generating key: %v", err)
		}
		keys[i] = ecdsaKey
	}

	bootstrapper, err := p2p.NewLibP2PHost(ctx, keys[0], 0)
	require.Nil(t, err)

	nodeA, err := p2p.NewLibP2PHost(ctx, keys[1], 0)
	require.Nil(t, err)

	nodeB, err := p2p.NewLibP2PHost(ctx, keys[2], 0)
	require.Nil(t, err)

	_, err = nodeA.Bootstrap(bootstrapAddresses(bootstrapper))
	require.Nil(t, err)

	_, err = nodeB.Bootstrap(bootstrapAddresses(bootstrapper))
	require.Nil(t, err)

	err = nodeA.WaitForBootstrap(2, 5*time.Second)
	require.Nil(t, err)

	err = nodeB.WaitForBootstrap(2, 5*time.Second)
	require.Nil(t, err)

	actorContext := actor.EmptyRootContext

	validTx := &services.AddBlockRequest{
		ObjectId: []byte("valid"),
	}

	networkPubsubA := NewNetworkPubSub(nodeA)
	networkPubsubB := NewNetworkPubSub(nodeB)

	topicName := "testpubsub"

	t.Run("regular broadcast", func(t *testing.T) {
		subscriber := actor.NewFuture(200 * time.Millisecond)
		ready := actor.NewFuture(1 * time.Second)
		parent := func(actCtx actor.Context) {
			switch msg := actCtx.Message().(type) {
			case *actor.Started:
				actCtx.Spawn(networkPubsubB.NewSubscriberProps(topicName))
				actCtx.Send(ready.PID(), true)
			case *services.AddBlockRequest:
				actCtx.Send(subscriber.PID(), msg)
			}
		}

		receiver, err := actorContext.SpawnNamed(actor.PropsFromFunc(parent), "pubsubtest-receiver-broadcast")
		require.Nil(t, err)
		defer receiver.Poison()

		_, err = ready.Result()
		require.Nil(t, err)
		time.Sleep(100 * time.Millisecond)

		err = networkPubsubA.Broadcast(topicName, validTx)
		require.Nil(t, err)

		resp, err := subscriber.Result()
		require.Nil(t, err)

		assert.Equal(t, validTx.ObjectId, resp.(*services.AddBlockRequest).ObjectId)
	})

	t.Run("validations", func(t *testing.T) {
		invalidTx := &services.AddBlockRequest{
			ObjectId: []byte("invalid"),
		}

		validator := func(ctx context.Context, p peer.ID, msg proto.Message) bool {
			return bytes.Equal(msg.(*services.AddBlockRequest).ObjectId, validTx.ObjectId)
		}

		err := networkPubsubA.RegisterTopicValidator(topicName, validator)
		require.Nil(t, err)
		defer networkPubsubA.UnregisterTopicValidator(topicName)

		err = networkPubsubB.RegisterTopicValidator(topicName, validator)
		require.Nil(t, err)
		defer networkPubsubB.UnregisterTopicValidator(topicName)

		subscriber := actor.NewFuture(200 * time.Millisecond)
		ready := actor.NewFuture(1 * time.Second)
		parent := func(actCtx actor.Context) {
			switch msg := actCtx.Message().(type) {
			case *actor.Started:
				actCtx.Spawn(networkPubsubB.NewSubscriberProps(topicName))
				actCtx.Send(ready.PID(), true)
			case *services.AddBlockRequest:
				actCtx.Send(subscriber.PID(), msg)
			}
		}

		receiver, err := actorContext.SpawnNamed(actor.PropsFromFunc(parent), "pubsubtest-receiver-validations")
		require.Nil(t, err)
		defer receiver.Poison()

		_, err = ready.Result()
		require.Nil(t, err)

		err = networkPubsubA.Broadcast(topicName, invalidTx)
		require.Nil(t, err)

		err = networkPubsubA.Broadcast(topicName, validTx)
		require.Nil(t, err)

		resp, err := subscriber.Result()
		require.Nil(t, err)

		// assert that even though we sent the invalidTx first, we still get the validTx back
		assert.Equal(t, validTx.ObjectId, resp.(*services.AddBlockRequest).ObjectId)
	})
}

func TestSimulatedBroadcaster(t *testing.T) {
	actorContext := actor.EmptyRootContext

	validTx := &services.AddBlockRequest{
		ObjectId: []byte("totaltest"),
	}

	simulatedPubSub := NewSimulatedPubSub()
	topicName := "testsimulatortopic"

	t.Run("regular broadcast", func(t *testing.T) {
		subscriber := actor.NewFuture(3 * time.Second)
		ready := actor.NewFuture(500 * time.Millisecond)
		parent := func(actCtx actor.Context) {
			switch msg := actCtx.Message().(type) {
			case *actor.Started:
				actCtx.Spawn(simulatedPubSub.NewSubscriberProps(topicName))
				time.Sleep(50 * time.Millisecond)
				actCtx.Send(ready.PID(), true)
			case *services.AddBlockRequest:
				actCtx.Send(subscriber.PID(), msg)
			default:
				middleware.Log.Debugw("parent received message type", "type", reflect.TypeOf(msg).String())
			}
		}

		receiver, err := actorContext.SpawnNamed(actor.PropsFromFunc(parent), "pubsubtest-simulator-broadcast")
		require.Nil(t, err)
		defer receiver.Poison()

		_, err = ready.Result()
		require.Nil(t, err)

		err = simulatedPubSub.Broadcast(topicName, validTx)
		require.Nil(t, err)

		resp, err := subscriber.Result()
		require.Nil(t, err)

		assert.Equal(t, validTx.ObjectId, resp.(*services.AddBlockRequest).ObjectId)
	})

	t.Run("validations", func(t *testing.T) {
		invalidTx := &services.AddBlockRequest{
			ObjectId: []byte("invalid"),
		}

		validator := func(ctx context.Context, p peer.ID, msg proto.Message) bool {
			return bytes.Equal(msg.(*services.AddBlockRequest).ObjectId, validTx.ObjectId)
		}

		err := simulatedPubSub.RegisterTopicValidator(topicName, validator)
		require.Nil(t, err)
		defer simulatedPubSub.UnregisterTopicValidator(topicName)

		subscriber := actor.NewFuture(200 * time.Millisecond)
		ready := actor.NewFuture(1 * time.Second)
		parent := func(actCtx actor.Context) {
			switch msg := actCtx.Message().(type) {
			case *actor.Started:
				actCtx.Spawn(simulatedPubSub.NewSubscriberProps(topicName))
				time.Sleep(50 * time.Millisecond)
				actCtx.Send(ready.PID(), true)
			case *services.AddBlockRequest:
				actCtx.Send(subscriber.PID(), msg)
			}
		}

		receiver, err := actorContext.SpawnNamed(actor.PropsFromFunc(parent), "pubsubtest-simulator-validations")
		require.Nil(t, err)
		defer receiver.Poison()

		_, err = ready.Result()
		require.Nil(t, err)

		err = simulatedPubSub.Broadcast(topicName, invalidTx)
		require.Nil(t, err)

		err = simulatedPubSub.Broadcast(topicName, validTx)
		require.Nil(t, err)

		resp, err := subscriber.Result()
		require.Nil(t, err)

		// assert that even though we sent the invalidTx first, we still get the validTx back
		assert.Equal(t, validTx.ObjectId, resp.(*services.AddBlockRequest).ObjectId)
	})
}

func bootstrapAddresses(bootstrapHost p2p.Node) []string {
	addresses := bootstrapHost.Addresses()
	for _, addr := range addresses {
		addrStr := addr.String()
		if strings.Contains(addrStr, "127.0.0.1") {
			return []string{addrStr}
		}
	}
	return nil
}
