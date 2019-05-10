package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"os"

	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	circuit "github.com/libp2p/go-libp2p-circuit"
	metrics "github.com/libp2p/go-libp2p-metrics"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/libp2p/go-libp2p"
	ma "github.com/multiformats/go-multiaddr"
)

const defaultPublicIP = "0.0.0.0"

type addressFactory func([]ma.Multiaddr) []ma.Multiaddr

type configFactory func(c *Config) error

type Config struct {
	RelayOpts            []circuit.RelayOpt
	EnableRelayHop       bool
	EnableAutoRelay      bool
	EnableBitSwap        bool
	PubSubRouter         string
	PubSubOptions        []pubsub.Option
	PrivateKey           *ecdsa.PrivateKey
	EnableNATMap         bool
	ListenAddrs          []string
	AddrFilters          []*net.IPNet
	Port                 int
	PublicIP             string
	DiscoveryNamespaces  []string
	AdditionalP2POptions []libp2p.Option
	DataStore            ds.Batching
	BandwidthReporter    metrics.Reporter
	Segmenter            []byte
	addressFactory       addressFactory
}

var defaultOptions = []configFactory{
	WithPubSubRouter("gossip"),
	WithPubSubOptions(pubsub.WithStrictSignatureVerification(false), pubsub.WithMessageSigning(false)),
	EnableNATMap(),
	WithListenIP("0.0.0.0", 0),
	WithBandWithReporter(metrics.NewBandwidthCounter()),
	WithDatastore(dsync.MutexWrap(ds.NewMapDatastore())),
}

func applyOptions(c *Config, opts ...configFactory) error {
	for _, factory := range opts {
		err := factory(c)
		if err != nil {
			return fmt.Errorf("error applying option: %v", err)
		}
	}
	return nil
}

func backwardsCompatibleConfig(key *ecdsa.PrivateKey, port int, useRelay bool) (*Config, error) {
	c := &Config{}
	opts := defaultOptions

	backwardsOpts := []configFactory{
		WithKey(key),
		WithDiscoveryNamespaces("tupelo-transaction-gossipers"),
		WithListenIP("0.0.0.0", port),
	}
	opts = append(opts, backwardsOpts...)

	if hostIP, ok := os.LookupEnv("TUPELO_PUBLIC_IP"); ok {
		opts = append(opts, WithExternalIP(hostIP, port))
	}

	if useRelay {
		opts = append(opts, WithRelayOpts(circuit.OptActive, circuit.OptHop))
	}

	err := applyOptions(c, opts...)
	if err != nil {
		return nil, fmt.Errorf("error applying opts: %v", err)
	}

	return c, nil
}

func WithAddrFilters(addrFilters []string) configFactory {
	return func(c *Config) error {
		addrFilterIPs := make([]*net.IPNet, len(addrFilters))
		for i, cidr := range addrFilters {
			net, err := stringToIPNet(cidr)
			if err != nil {
				return fmt.Errorf("error getting stringToIPnet: %v", err)
			}
			addrFilterIPs[i] = net
		}
		c.AddrFilters = addrFilterIPs
		return nil
	}
}

func WithDiscoveryNamespaces(namespaces ...string) configFactory {
	return func(c *Config) error {
		c.DiscoveryNamespaces = namespaces
		return nil
	}
}

func WithSegmenter(secret []byte) configFactory {
	return func(c *Config) error {
		c.Segmenter = secret
		return nil
	}
}

func WithBandWithReporter(reporter metrics.Reporter) configFactory {
	return func(c *Config) error {
		c.BandwidthReporter = reporter
		return nil
	}
}

func WithPubSubOptions(opts ...pubsub.Option) configFactory {
	return func(c *Config) error {
		c.PubSubOptions = opts
		return nil
	}
}

func WithDatastore(store ds.Batching) configFactory {
	return func(c *Config) error {
		c.DataStore = store
		return nil
	}
}

func WithRelayOpts(opts ...circuit.RelayOpt) configFactory {
	return func(c *Config) error {
		c.RelayOpts = opts
		return nil
	}
}

func WithKey(key *ecdsa.PrivateKey) configFactory {
	return func(c *Config) error {
		c.PrivateKey = key
		return nil
	}
}

func WithBitswap(enabled bool) configFactory {
	return func(c *Config) error {
		c.EnableBitSwap = enabled
		return nil
	}
}

func WithListenIP(ip string, port int) configFactory {
	return func(c *Config) error {
		c.Port = port
		c.ListenAddrs = []string{fmt.Sprintf("/ip4/%s/tcp/%d", ip, c.Port)}
		return nil
	}
}

var supportedGossipTypes = map[string]bool{"gossip": true, "random": true, "floodsub": true}

func WithPubSubRouter(routeType string) configFactory {
	return func(c *Config) error {
		if _, ok := supportedGossipTypes[routeType]; !ok {
			return fmt.Errorf("%s is an unsupported gossip type", routeType)
		}
		c.PubSubRouter = routeType
		return nil
	}
}

func EnableNATMap() configFactory {
	return func(c *Config) error {
		c.EnableNATMap = true
		return nil
	}
}

func WithExternalIP(ip string, port int) configFactory {
	return func(c *Config) error {
		extMaddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", ip, c.Port))

		if err != nil {
			return fmt.Errorf("Error creating multiaddress: %v", err)
		}

		addressFactory := func(addrs []ma.Multiaddr) []ma.Multiaddr {
			return append(addrs, extMaddr)
		}

		c.addressFactory = addressFactory
		return nil
	}
}

func stringToIPNet(str string) (*net.IPNet, error) {
	_, n, err := net.ParseCIDR(str)
	return n, fmt.Errorf("error parsing %s: %v", str, err)
}
