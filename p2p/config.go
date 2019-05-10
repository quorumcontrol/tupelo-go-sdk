package p2p

import (
	"crypto/ecdsa"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p"
	ma "github.com/multiformats/go-multiaddr"
)

const defaultPublicIP = "0.0.0.0"

type addressFactory func([]ma.Multiaddr) []ma.Multiaddr

type configFactory func(c *Config) error

type Config struct {
	EnableRelayActive    bool
	EnableRelayHop       bool
	EnableAutoRelay      bool
	EnableBitSwap        bool
	PubSubRouter         string
	PrivateKey           *ecdsa.PrivateKey
	EnableNATMap         bool
	ListenAddrs          []string
	AddrFilters          []string
	Port                 int
	PublicIP             string
	Discovery            []string
	AdditionalP2POptions []libp2p.Option
	addressFactory       addressFactory
}

var defaultOptions = []configFactory{
	WithPubSubRouter("gossip"),
	EnableNATMap(),
	WithListenIP("0.0.0.0", 0),
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
	opts = append(opts, WithListenIP("0.0.0.0", port))
	opts = append(opts, WithKey(key))

	if hostIP, ok := os.LookupEnv("TUPELO_PUBLIC_IP"); ok {
		opts = append(opts, WithExternalIP(hostIP, port))
	}

	if useRelay {
		opts = append(opts, WithRelayActive())
		opts = append(opts, WithRelayHop())
	}

	err := applyOptions(c, opts...)
	if err != nil {
		return nil, fmt.Errorf("error applying opts: %v", err)
	}

	return c, nil
}

func WithRelayActive() configFactory {
	return func(c *Config) error {
		c.EnableRelayActive = true
		return nil
	}
}

func WithRelayHop() configFactory {
	return func(c *Config) error {
		c.EnableRelayHop = true
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
		c.ListenAddrs = []string{fmt.Sprintf("/ip4/%s/%d", ip, c.Port)}
		return nil
	}
}

func WithPubSubRouter(routType string) configFactory {
	return func(c *Config) error {
		c.PubSubRouter = routType
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
