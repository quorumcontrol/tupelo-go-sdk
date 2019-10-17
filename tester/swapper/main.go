package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"time"

	logging "github.com/ipfs/go-log"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
)

var logger = logging.Logger("swapper-main")

func getNode(ctx context.Context) (p2p.Node, *p2p.BitswapPeer, error) {
	cm := connmgr.NewConnManager(15, 900, 20*time.Second)

	opts := []p2p.Option{
		p2p.WithDiscoveryNamespaces("tupelo-transaction-gossipers"),
		p2p.WithListenIP("0.0.0.0", 34001),
		p2p.WithLibp2pOptions(libp2p.ConnectionManager(cm)),
	}

	return p2p.NewHostAndBitSwapPeer(ctx, opts...)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n, peer, err := getNode(ctx)
	if err != nil {
		panic(err)
	}

	c, err := n.Bootstrap([]string{"/ip4/172.16.245.10/tcp/34001/ipfs/16Uiu2HAm3TGSEKEjagcCojSJeaT5rypaeJMKejijvYSnAjviWwV5"})
	if err != nil {
		panic(err)
	}
	defer c.Close()

	err = n.WaitForBootstrap(1, 5*time.Second)
	if err != nil {
		panic(err)
	}

	sw := &safewrap.SafeWrap{}

	var doPuts = flag.Bool("puts", false, "actually put the blocks?")
	flag.Parse()

	if *doPuts {
		logging.SetLogLevel("dht", "debug")

		logger.Info("-------- starting to put blocks")
		for i := 0; i < 75000; i++ {
			n := sw.WrapObject(map[string]string{"hello": "bitswap" + strconv.Itoa(i)})
			if sw.Err != nil {
				panic("error wrapping")
			}

			swapCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

			err = peer.Add(swapCtx, n)
			if err != nil {
				panic("error storeing")
			}
			cancel()
		}
		logger.Info("--------- finished put blocks")
	}

	fmt.Println("swapper launched")
	select {}
}
