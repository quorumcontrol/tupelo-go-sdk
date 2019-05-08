package p2p

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func libP2PNodeGenerator(ctx context.Context, t *testing.T) Node {
	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	host, err := NewLibP2PHost(ctx, key, 0)

	require.Nil(t, err)
	require.NotNil(t, host)

	return host
}

func TestHost(t *testing.T) {
	NodeTests(t, libP2PNodeGenerator)
}

func TestP2pBroadcastIp(t *testing.T) {
	startEnv, startEnvOk := os.LookupEnv("TUPELO_PUBLIC_IP")
	defer func() {
		if startEnvOk {
			os.Setenv("TUPELO_PUBLIC_IP", startEnv)
		} else {
			os.Unsetenv("TUPELO_PUBLIC_IP")
		}
	}()
	os.Setenv("TUPELO_PUBLIC_IP", "1.1.1.1")
	testHost := libP2PNodeGenerator(context.Background(), t)
	found := false
	for _, addr := range testHost.Addresses() {
		if strings.HasPrefix(addr.String(), "/ip4/1.1.1.1/") {
			found = true
		}
	}
	require.True(t, found)
}

func TestWrap(t *testing.T) {
	NodeTests(t, func(ctx context.Context, t *testing.T) Node {
		key, err := crypto.GenerateKey()
		require.Nil(t, err)

		priv, err := p2pPrivateFromEcdsaPrivate(key)
		require.Nil(t, err)

		opts := []libp2p.Option{
			libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", 0)),
			libp2p.Identity(priv),
			libp2p.DefaultTransports,
			libp2p.DefaultMuxers,
			libp2p.DefaultSecurity,
			libp2p.NATPortMap(),
		}

		basicHost, err := libp2p.New(ctx, opts...)
		require.Nil(t, err)
		dstore := dsync.MutexWrap(ds.NewMapDatastore())
		dht := dht.NewDHT(ctx, basicHost, dstore)
		routedHost := rhost.Wrap(basicHost, dht)

		node, err := Wrap(ctx, routedHost, dht)
		require.Nil(t, err)
		return node
	})
}

func TestUnmarshal31ByteKey(t *testing.T) {
	// we kept having failures that looked like lack of entropy, but
	// were instead just a normal case where a 31 byte big.Int was generated as a
	// private key from the crypto library. This is one such key:
	keyHex := "0x0092f4d28a01ad9432b4cb9ffb1ecf5628bff465a231b86c9a5b739ead0b3bb5"
	keyBytes, err := hexutil.Decode(keyHex)
	require.Nil(t, err)
	key, err := crypto.ToECDSA(keyBytes)
	require.Nil(t, err)
	libP2PKey, err := p2pPrivateFromEcdsaPrivate(key)
	require.Nil(t, err)

	libP2PPub, err := libP2PKey.GetPublic().Raw()
	require.Nil(t, err)

	assert.Equal(t, crypto.CompressPubkey(&key.PublicKey), libP2PPub)
}
