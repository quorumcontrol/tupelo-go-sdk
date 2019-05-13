package p2p

import (
	"context"
	"fmt"
	"time"

	"sync/atomic"

	"github.com/AsynkronIT/protoactor-go/eventstream"
	discovery "github.com/libp2p/go-libp2p-discovery"
	inet "github.com/libp2p/go-libp2p-net"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

const maxConnected = 300

const (
	// EventPeerConnected is emitted to the eventstream
	// whenever a new peer is found
	EventPeerConnected int = iota
)

type tupeloDiscoverer struct {
	namespace  string
	host       *LibP2PHost
	discoverer *discovery.RoutingDiscovery
	connected  uint64
	events     *eventstream.EventStream
}

type DiscoveryEvent struct {
	Namespace string
	Connected uint64
	EventType int
}

func newTupeloDiscoverer(h *LibP2PHost, namespace string) *tupeloDiscoverer {
	return &tupeloDiscoverer{
		namespace:  namespace,
		host:       h,
		discoverer: discovery.NewRoutingDiscovery(h.routing),
		events:     &eventstream.EventStream{},
	}
}

func (td *tupeloDiscoverer) doDiscovery(ctx context.Context) error {
	if err := td.constantlyAdvertise(ctx); err != nil {
		return fmt.Errorf("error advertising: %v", err)
	}
	if err := td.findPeers(ctx); err != nil {
		return fmt.Errorf("error finding peers: %v", err)
	}
	return nil
}

func (td *tupeloDiscoverer) findPeers(ctx context.Context) error {
	peerChan, err := td.discoverer.FindPeers(ctx, td.namespace)
	if err != nil {
		return fmt.Errorf("error findPeers: %v", err)
	}

	go func() {
		for peerInfo := range peerChan {
			td.handleNewPeerInfo(ctx, peerInfo)
		}
	}()
	return nil
}

func (td *tupeloDiscoverer) handleNewPeerInfo(ctx context.Context, p pstore.PeerInfo) {
	if p.ID == "" {
		return // empty id
	}

	host := td.host.host

	if host.Network().Connectedness(p.ID) == inet.Connected {
		return // we are already connected
	}

	connected := host.Network().Peers()
	if len(connected) > maxConnected {
		return // we already are connected to more than we need
	}

	log.Debugf("new peer: %s", p.ID)

	// do the connection async because connect can hang
	go func() {
		// not actually positive that TTL is correct, but it seemed the most appropriate
		host.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.ProviderAddrTTL)
		if err := host.Connect(ctx, p); err != nil {
			log.Errorf("error connecting to  %s %v: %v", p.ID, p, err)
		}
		numConnected := uint64(len(connected))
		atomic.StoreUint64(&td.connected, numConnected)
		td.events.Publish(&DiscoveryEvent{
			Namespace: td.namespace,
			Connected: numConnected,
			EventType: EventPeerConnected,
		})
	}()
}

func (td *tupeloDiscoverer) constantlyAdvertise(ctx context.Context) error {
	dur, err := td.discoverer.Advertise(ctx, td.namespace)
	if err != nil {
		return err
	}
	go func() {
		after := time.After(dur)
		select {
		case <-ctx.Done():
			return
		case <-after:
			if err := td.constantlyAdvertise(ctx); err != nil {
				log.Errorf("error constantly advertising: %v", err)
			}
		}
	}()
	return nil
}
