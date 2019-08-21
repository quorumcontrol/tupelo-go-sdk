package p2p

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
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

func TestWithExternalAddrs(t *testing.T) {
	c := &Config{}
	err := applyOptions(c, WithExternalIP("1.1.1.1", 53))
	require.Nil(t, err)
	err = applyOptions(c, WithWebSocketExternalIP("1.1.1.1", 80))
	require.Nil(t, err)
	err = applyOptions(c, WithExternalIP("2.2.2.2", 53))
	require.Nil(t, err)
	err = applyOptions(c, WithWebSocketExternalIP("2.2.2.2", 80))
	require.Nil(t, err)
	err = applyOptions(c, WithWebRTCExternalIP("3.3.3.3", 90))
	require.Nil(t, err)
	require.ElementsMatch(t, c.ExternalAddrs, []string{
		"/ip4/1.1.1.1/tcp/53",
		"/ip4/1.1.1.1/tcp/80/ws",
		"/ip4/2.2.2.2/tcp/53",
		"/ip4/2.2.2.2/tcp/80/ws",
		"/ip4/3.3.3.3/tcp/90/http/p2p-webrtc-direct",
	})
}

func TestNewRelayLibP2PHost(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h, err := NewRelayLibP2PHost(ctx, key, 0)
	require.Nil(t, err)
	require.NotNil(t, h)
}

func TestNewHostFromOptions(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	t.Run("it works with no options", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		h, err := NewHostFromOptions(ctx)
		require.Nil(t, err)
		require.NotNil(t, h)
	})

	t.Run("it works with more esoteric options", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cm := connmgr.NewConnManager(20, 100, 20*time.Second)

		h, err := NewHostFromOptions(
			ctx,
			WithKey(key),
			WithAutoRelay(true),
			WithSegmenter([]byte("my secret password")),
			WithPubSubRouter("floodsub"),
			WithRelayOpts(circuit.OptHop),
			WithLibp2pOptions(libp2p.ConnectionManager(cm)),
			WithClientOnlyDHT(true),
			WithWebRTC(0),
			WithWebSockets(0),
		)
		require.Nil(t, err)
		require.NotNil(t, h)
		var addrs string
		for _, addr := range h.Addresses() {
			addrs = addrs + ";" + addr.String()
		}
		require.Truef(t, strings.Index(addrs, "p2p-webrtc-direct") > 0, "addrs %s did not include p2p-webrtc-direct", addrs)
		require.Truef(t, strings.Index(addrs, "ws/ipfs") > 0, "addrs %s did not include ws/ipfs", addrs)
		cancel()
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

func TestWaitForDiscovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h1, err := NewHostFromOptions(ctx, WithDiscoveryNamespaces("one", "two"))
	require.Nil(t, err)

	h2, err := NewHostFromOptions(ctx, WithDiscoveryNamespaces("one"))
	require.Nil(t, err)

	_, err = h1.Bootstrap(bootstrapAddresses(h2))
	require.Nil(t, err)

	_, err = h2.Bootstrap(bootstrapAddresses(h1))
	require.Nil(t, err)

	err = h1.WaitForBootstrap(1, 10*time.Second)
	require.Nil(t, err)

	// one should already be fine and not wait at all (because of bootstrap)
	err = h1.WaitForDiscovery("one", 1, 2*time.Second)
	require.Nil(t, err)
}

func TestNewDiscoverers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h1, err := NewHostFromOptions(ctx)
	require.Nil(t, err)

	h2, err := NewHostFromOptions(ctx)
	require.Nil(t, err)

	_, err = h1.Bootstrap(bootstrapAddresses(h2))
	require.Nil(t, err)

	err = h1.WaitForBootstrap(1, 10*time.Second)
	require.Nil(t, err)

	ns := "totally-new"

	err = h1.StartDiscovery(ns)
	require.Nil(t, err)

	defer h1.StopDiscovery(ns)

	err = h2.StartDiscovery(ns)
	require.Nil(t, err)
	defer h2.StopDiscovery(ns)

	err = h2.WaitForDiscovery(ns, 1, 2*time.Second)
	require.Nil(t, err)
}
