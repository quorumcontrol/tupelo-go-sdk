package p2p

import (
	"context"

	"github.com/ipfs/go-bitswap"
	"github.com/ipfs/go-bitswap/network"
	"github.com/ipfs/go-blockservice"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/ipfs/go-merkledag"

	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	dag "github.com/ipfs/go-merkledag"

	blockstore "github.com/ipfs/go-ipfs-blockstore"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

func init() {
	ipld.Register(cid.DagProtobuf, dag.DecodeProtobufBlock)
	ipld.Register(cid.Raw, dag.DecodeRawBlock)
	ipld.Register(cid.DagCBOR, cbor.DecodeBlock) // need to decode CBOR
}

// Peer is an IPFS-Lite peer. It provides a DAG service that can fetch and put
// blocks from/to the IPFS network.
type BitswapPeer struct {
	ipld.DAGService

	ctx context.Context

	bstore blockstore.Blockstore
	host   host.Host
	dht    *dht.IpfsDHT
}

// Create a new block-swapping peer from an existing *LibP2PHost
func NewBitswapPeer(ctx context.Context, host *LibP2PHost) (*BitswapPeer, error) {
	bs := blockstore.NewBlockstore(host.datastore)
	bs = blockstore.NewIdStore(bs)
	cachedbs, err := blockstore.CachedBlockstore(ctx, bs, blockstore.DefaultCacheOpts())
	if err != nil {
		return nil, err
	}

	bswapnet := network.NewFromIpfsHost(host.host, host.routing)
	bswap := bitswap.New(ctx, bswapnet, cachedbs)
	bserv := blockservice.New(cachedbs, bswap)

	dags := merkledag.NewDAGService(bserv)
	return &BitswapPeer{
		DAGService: dags,
		bstore:     cachedbs,
	}, nil
}

// Session returns a session-based NodeGetter.
func (bp *BitswapPeer) Session(ctx context.Context) ipld.NodeGetter {
	ng := merkledag.NewSession(ctx, bp.DAGService)
	if ng == bp.DAGService {
		log.Warning("DAGService does not support sessions")
	}
	return ng
}

// BlockStore offers access to the blockstore underlying the Peer's DAGService.
func (bp *BitswapPeer) BlockStore() blockstore.Blockstore {
	return bp.bstore
}

// HasBlock returns whether a given block is available locally. It is
// a shorthand for .Blockstore().Has().
func (bp *BitswapPeer) HasBlock(c cid.Cid) (bool, error) {
	return bp.BlockStore().Has(c)
}
