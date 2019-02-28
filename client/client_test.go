// +build integration

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/storage"
	"github.com/quorumcontrol/tupelo-go-client/bls"
	"github.com/quorumcontrol/tupelo-go-client/consensus"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/testhelpers"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-client/p2p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type publicKeySet struct {
	BlsHexPublicKey   string `json:"blsHexPublicKey,omitempty"`
	EcdsaHexPublicKey string `json:"ecdsaHexPublicKey,omitempty"`
	PeerIDBase58Key   string `json:"peerIDBase58Key,omitempty"`
}

func loadSignerKeys() ([]*publicKeySet, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("No caller information")
	}
	jsonBytes, err := ioutil.ReadFile(path.Join(path.Dir(filename), "test-signer-keys/public-keys.json"))
	if err != nil {
		return nil, err
	}
	var keySet []*publicKeySet
	if err := json.Unmarshal(jsonBytes, &keySet); err != nil {
		return nil, err
	}

	return keySet, nil
}

func setupNotaryGroup(ctx context.Context) (*types.NotaryGroup, error) {
	remote.Start()
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("error generating key: %s", err)
	}
	p2pHost, err := p2p.NewLibP2PHost(ctx, key, 0)
	if err != nil {
		return nil, fmt.Errorf("error setting up p2p host: %s", err)
	}
	p2pHost.Bootstrap(p2p.BootstrapNodes())
	p2pHost.WaitForBootstrap(1, 15*time.Second)
	remote.NewRouter(p2pHost)

	keys, err := loadSignerKeys()
	if err != nil {
		return nil, err
	}
	group := types.NewNotaryGroup("hardcodedprivatekeysareunsafe")
	for _, keySet := range keys {
		ecdsaBytes := hexutil.MustDecode(keySet.EcdsaHexPublicKey)
		verKeyBytes := hexutil.MustDecode(keySet.BlsHexPublicKey)
		ecdsaPubKey, err := crypto.UnmarshalPubkey(ecdsaBytes)
		if err != nil {
			return nil, err
		}
		signer := types.NewRemoteSigner(ecdsaPubKey, bls.BytesToVerKey(verKeyBytes))
		group.AddSigner(signer)
	}
	group.SetupAllRemoteActors(&key.PublicKey)

	return group, nil
}

func TestClientSendTransaction(t *testing.T) {
	// ts := testnotarygroup.NewTestSet(t, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	client := New(ng)
	defer client.Stop()

	trans := testhelpers.NewValidTransaction(t)
	err = client.SendTransaction(ng.GetRandomSigner(), &trans)
	require.Nil(t, err)
}

func TestClientSubscribe(t *testing.T) {
	// ts := testnotarygroup.NewTestSet(t, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	trans := testhelpers.NewValidTransaction(t)
	client := New(ng)
	defer client.Stop()

	ch, err := client.Subscribe(ng.GetRandomSigner(), string(trans.ObjectID), 5*time.Second)
	require.Nil(t, err)

	err = client.SendTransaction(ng.GetRandomSigner(), &trans)
	require.Nil(t, err)

	resp := <-ch
	require.NotNil(t, ch)
	assert.Equal(t, resp.Signature.NewTip, trans.NewTip)
}

func TestPlayTransactions(t *testing.T) {
	// ts := testnotarygroup.NewTestSet(t, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ng, err := setupNotaryGroup(ctx)
	require.Nil(t, err)

	client := New(ng)
	defer client.Stop()

	treeKey, err := crypto.GenerateKey()
	require.Nil(t, err)
	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	chain, err := consensus.NewSignedChainTree(treeKey.PublicKey, nodeStore)

	var remoteTip string
	if !chain.IsGenesis() {
		remoteTip = chain.Tip().String()
	}

	resp, err := client.PlayTransactions(chain, treeKey, remoteTip, []*chaintree.Transaction{
		{
			Type: "SET_DATA",
			Payload: map[string]string{
				"path":  "down/in/the/thing",
				"value": "sometestvalue",
			},
		},
	})
	require.Nil(t, err)
	assert.Equal(t, resp.Tip.Bytes(), chain.Tip().Bytes())

	t.Run("works on 2nd set", func(t *testing.T) {
		resp, err := client.PlayTransactions(chain, treeKey, chain.Tip().String(), []*chaintree.Transaction{
			{
				Type: "SET_DATA",
				Payload: map[string]string{
					"path":  "down/in/the/thing",
					"value": "sometestvalue",
				},
			},
		})
		require.Nil(t, err)
		assert.Equal(t, resp.Tip.Bytes(), chain.Tip().Bytes())

		// and works a third time
		resp, err = client.PlayTransactions(chain, treeKey, chain.Tip().String(), []*chaintree.Transaction{
			{
				Type: "SET_DATA",
				Payload: map[string]string{
					"path":  "down/in/the/thing",
					"value": "sometestvalue",
				},
			},
		})
		require.Nil(t, err)
		assert.Equal(t, resp.Tip.Bytes(), chain.Tip().Bytes())
	})
}
