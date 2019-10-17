package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
)

const destKey = "0xe2c0b170c56ff0bf08c7376f1ca4bd5cbe481a85f6cdb3de609863e25dce613a"

func getKey() (*ecdsa.PrivateKey, error) {
	destKeyBytes, err := hexutil.Decode(destKey)
	if err != nil {
		return nil, fmt.Errorf("error decoding dest key: %v", err)
	}
	ecdsaPrivate, err := crypto.ToECDSA(destKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal ECDSA private key: %v", err)
	}
	return ecdsaPrivate, nil
}

func getNode(ctx context.Context) (p2p.Node, error) {
	key, err := getKey()
	if err != nil {
		return nil, err
	}
	cm := connmgr.NewConnManager(4915, 7372, 30*time.Second)

	opts := []p2p.Option{
		p2p.WithKey(key),
		p2p.WithDiscoveryNamespaces("tupelo-transaction-gossipers"),
		p2p.WithListenIP("0.0.0.0", 34001),
		p2p.WithLibp2pOptions(libp2p.ConnectionManager(cm)),
		p2p.WithRelayOpts(circuit.OptHop),
	}

	return p2p.NewHostFromOptions(ctx, opts...)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := getNode(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("bootstrapper launched")
	select {}
}
