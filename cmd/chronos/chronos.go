package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
	"go-chronos/core"
	"go-chronos/node"
	"go-chronos/utils"
	"time"
)

func main() {
	flag.Parse()

	if help {
		flag.Usage()
		return
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 数据库、节点的启动

	db, err := utils.NewLevelDB(datadir)

	if err != nil {
		log.WithField("error", err).Errorln("Create or load database failed.")
		return
	}

	chain := core.NewBlockchain(db)
	txPool := core.NewTxPool()
	hConfig := node.HandlerConfig{
		TxPool: txPool,
		Chain:  chain,
	}

	_, err = node.NewHandler(&hConfig)
	if err != nil {
		log.WithField("error", err).Errorln("Create handler failed.")
		return
	}

	// 网络部分的启动

	localMultiAddr, err := multiaddr.NewMultiaddr(
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
	)
	if err != nil {
		log.WithField("error", err).Errorln("Convert multiple address failed")
		return
	}

	host, err := libp2p.New(
		libp2p.ListenAddrs(localMultiAddr),
	)
	if err != nil {
		log.WithField("error", err).Errorln("Create local host failed.")
	}

	host.SetStreamHandler(node.ProtocolId, node.HandleStream)
	log.Infof("Node address: /ip4/127.0.0.1/tcp/%v/p2p/%s", port, host.ID().String())

	var kdht *dht.IpfsDHT

	if bootstrap == "" {
		kdht, err = NewKDHT(ctx, host, []multiaddr.Multiaddr{})
		if err != nil {
			log.WithField("error", err).Errorln("Create kademlia server failed.")
			return
		}
	} else {
		maddr, err := multiaddr.NewMultiaddr(bootstrap)

		if err != nil {
			log.WithField("error", err).Errorln("Covert address to multiple addrerss failed.")
			return
		}
		kdht, err = NewKDHT(ctx, host, []multiaddr.Multiaddr{maddr})
		if err != nil {
			log.WithField("error", err).Errorln("Create kademlia server failed.")
			return
		}
	}

	go node.Discover(ctx, host, kdht, "Chronos network.")

	if genesis {
		log.Infof("Create genesis block after 10s...")
		go func() {
			ticker := time.NewTicker(10 * time.Second)

			select {
			case <-ticker.C:
				log.Infof("Create genesis block.")
				chain.NewGenesisBlock()
			}
		}()
	}

	select {}
}
