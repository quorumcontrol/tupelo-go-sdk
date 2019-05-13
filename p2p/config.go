package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	circuit "github.com/libp2p/go-libp2p-circuit"
	metrics "github.com/libp2p/go-libp2p-metrics"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/libp2p/go-libp2p"
	ma "github.com/multiformats/go-multiaddr"
)

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

// This is a function, because we want to return a new datastore each time
func defaultOptions() []configFactory {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Errorf("unable to generate a new key"))
	}
	return []configFactory{
		WithKey(key),
		WithPubSubRouter("gossip"),
		WithPubSubOptions(pubsub.WithStrictSignatureVerification(false), pubsub.WithMessageSigning(false)),
		WithNATMap(),
		WithListenIP("0.0.0.0", 0),
		WithBandwidthReporter(metrics.NewBandwidthCounter()),
		WithDatastore(dsync.MutexWrap(ds.NewMapDatastore())),
	}
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
	opts := defaultOptions()

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

// WithAddrFilters takes a string of cidr addresses (0.0.0.0/32) that will not
// be dialed by swarm.
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

// WithAutoRelay enabled AutoRelay in the config, defaults to false
func WithAutoRelay(enabled bool) configFactory {
	return func(c *Config) error {
		c.EnableAutoRelay = enabled
		return nil
	}
}

// WithDiscoveryNamespaces enables discovery of all of the passed in namespaces
// discovery is part of bootstrap, defaults to empty
func WithDiscoveryNamespaces(namespaces ...string) configFactory {
	return func(c *Config) error {
		c.DiscoveryNamespaces = namespaces
		return nil
	}
}

// WithSegmenter enables the secret on libp2p in order to make sure
// that this network does not combine with another. Default is off.
func WithSegmenter(secret []byte) configFactory {
	return func(c *Config) error {
		c.Segmenter = secret
		return nil
	}
}

// WithBandwidthReporter sets the bandwidth reporter,
// defaults to a new metrics.Reporter
func WithBandwidthReporter(reporter metrics.Reporter) configFactory {
	return func(c *Config) error {
		c.BandwidthReporter = reporter
		return nil
	}
}

// WithPubSubOptions sets pubsub options.
// Defaults to pubsub.WithStrictSignatureVerification(false), pubsub.WithMessageSigning(false)
func WithPubSubOptions(opts ...pubsub.Option) configFactory {
	return func(c *Config) error {
		c.PubSubOptions = opts
		return nil
	}
}

// WithDatastore sets the datastore used by the host
// defaults to in-memory map store.
func WithDatastore(store ds.Batching) configFactory {
	return func(c *Config) error {
		c.DataStore = store
		return nil
	}
}

// WithRelayOpts turns on relay and sets the options
// defaults to empty (and relay off).
func WithRelayOpts(opts ...circuit.RelayOpt) configFactory {
	return func(c *Config) error {
		c.RelayOpts = opts
		return nil
	}
}

// WithKey is the identity key of the host, if not set it
// the default options will generate you a new key
func WithKey(key *ecdsa.PrivateKey) configFactory {
	return func(c *Config) error {
		c.PrivateKey = key
		return nil
	}
}

// WithListenIP sets the listen IP, defaults to 0.0.0.0/0
func WithListenIP(ip string, port int) configFactory {
	return func(c *Config) error {
		c.Port = port
		c.ListenAddrs = []string{fmt.Sprintf("/ip4/%s/tcp/%d", ip, c.Port)}
		return nil
	}
}

var supportedGossipTypes = map[string]bool{"gossip": true, "random": true, "floodsub": true}

// WithPubSubRouter sets the router type of the pubsub. Supported is: gossip, random, floodsub
// defaults to gossip
func WithPubSubRouter(routeType string) configFactory {
	return func(c *Config) error {
		if _, ok := supportedGossipTypes[routeType]; !ok {
			return fmt.Errorf("%s is an unsupported gossip type", routeType)
		}
		c.PubSubRouter = routeType
		return nil
	}
}

// WithNATMap enables nat mapping
func WithNATMap() configFactory {
	return func(c *Config) error {
		c.EnableNATMap = true
		return nil
	}
}

// WithExternalIP sets an arbitrary ip/port for broadcasting to swarm
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
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", str, err)
	}
	return n, nil
}