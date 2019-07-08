package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
)

// Standalone server for testing libp2p-js / libp2p-go interop

func startServer(ctx context.Context) error {
	key, err := crypto.GenerateKey()
	if err != nil {
		return errors.Wrap(err, "error generating key")
	}

	opts := []p2p.Option{
		p2p.WithKey(key),
		p2p.WithWebSockets(true),
	}

	host, err := p2p.NewHostFromOptions(ctx, opts...)
	if err != nil {
		return errors.Wrap(err, "error creating host")
	}

	fmt.Println("standalone node running at:")
	for _, addr := range host.Addresses() {
		fmt.Println(addr)
	}

	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := startServer(ctx)
	if err != nil {
		panic(err)
	}
	select {}
}
