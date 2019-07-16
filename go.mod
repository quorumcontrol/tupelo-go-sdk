module github.com/quorumcontrol/tupelo-go-sdk

go 1.12

require (
	github.com/AsynkronIT/protoactor-go v0.0.0-20190318154652-aa1aa20c2fa0
	github.com/BurntSushi/toml v0.3.1
	github.com/avast/retry-go v2.3.0+incompatible
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/elastic/go-sysinfo v1.0.1 // indirect
	github.com/ethereum/go-ethereum v1.8.25
	github.com/gogo/protobuf v1.2.1
	github.com/hashicorp/golang-lru v0.5.1
	github.com/ipfs/go-bitswap v0.1.5
	github.com/ipfs/go-blockservice v0.1.0
	github.com/ipfs/go-cid v0.0.2
	github.com/ipfs/go-datastore v0.0.5
	github.com/ipfs/go-ipfs-blockstore v0.0.1
	github.com/ipfs/go-ipld-cbor v1.5.1-0.20190302174746-59d816225550
	github.com/ipfs/go-ipld-format v0.0.2
	github.com/ipfs/go-log v0.0.1
	github.com/ipfs/go-merkledag v0.1.0
	github.com/libp2p/go-libp2p v0.2.0
	github.com/libp2p/go-libp2p-circuit v0.1.0
	github.com/libp2p/go-libp2p-connmgr v0.1.0
	github.com/libp2p/go-libp2p-core v0.0.6
	github.com/libp2p/go-libp2p-discovery v0.1.0
	github.com/libp2p/go-libp2p-kad-dht v0.1.1
	github.com/libp2p/go-libp2p-pnet v0.1.0
	github.com/libp2p/go-libp2p-pubsub v0.1.0
	github.com/libp2p/go-libp2p-swarm v0.1.1
	github.com/multiformats/go-multiaddr v0.0.4
	github.com/opentracing/opentracing-go v1.1.0
	github.com/prometheus/client_golang v0.9.3
	github.com/quorumcontrol/chaintree v0.0.0-20190701175144-f8f44c3e6d4b
	github.com/quorumcontrol/messages/build/go v0.0.0-20190716095704-9acdbae78c93
	github.com/santhosh-tekuri/jsonschema v1.2.4 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/uber-go/atomic v1.3.2 // indirect
	github.com/uber/jaeger-client-go v2.15.0+incompatible
	github.com/uber/jaeger-lib v1.5.0
	go.dedis.ch/kyber/v3 v3.0.2
	go.elastic.co/apm/module/apmot v1.3.0
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.9.1
)

replace github.com/libp2p/go-libp2p-pubsub v0.1.0 => github.com/quorumcontrol/go-libp2p-pubsub v0.0.4-0.20190528094025-e4e719f73e7a
