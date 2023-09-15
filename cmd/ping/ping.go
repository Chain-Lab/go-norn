package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
	"go-chronos/node"
	"go-chronos/p2p"
)

// !! 这部分代码只是初期测试 kademlia 使用，由于后续底层代码结构变动
// 该部分的代码已经不能完成测试

func NewKDHT(ctx context.Context, host host.Host, bootstrapPeers []multiaddr.Multiaddr) (*dht.IpfsDHT, error) {
	// dht 的配置项
	var options []dht.Option

	// 如果没有引导节点，以服务器模式 ModeServer 启动
	if len(bootstrapPeers) == 0 {
		options = append(options, dht.Mode(dht.ModeServer))
		//log.Infoln("Start node as a bootstrap server.")
	}

	// 生成一个 DHT 实例
	kdht, err := dht.New(ctx, host, options...)
	if err != nil {
		return nil, err
	}

	// 启动 DHT 服务
	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	// 遍历引导节点数组并尝试连接
	for _, peerAddr := range bootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		if err := host.Connect(ctx, *peerinfo); err != nil {
			log.Printf("Error while connecting to node %q: %-v", peerinfo, err)
			continue
		} else {
			s, err := host.NewStream(ctx, peerinfo.ID, "/ping/1.0.0")

			if err != nil {
				log.WithField("error", err).Debugln("Create new stream error.")
				continue
			}
			p, err := p2p.NewPeer(peerinfo.ID, &s, nil)

			go p.Run()
			log.Printf("Connection established with bootstrap node: %q", *peerinfo)
		}
	}

	return kdht, nil
}

func main() {
	log.SetLevel(log.InfoLevel)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sourcePort := flag.Int("sp", 0, "Source port number")
	dest := flag.String("d", "", "Destination multiaddr string")

	flag.Parse()

	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *sourcePort))
	host, err := libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
	)

	host.SetStreamHandler("/ping/1.0.0", node.HandleStream)

	//log.Infof("Node address: /ip4/127.0.0.1/tcp/%v/p2p/%s", *sourcePort, host.ID().String())

	if err != nil {
		log.WithField("error", err).Errorln("Create p2p host failed.")
		return
	}

	var dht *dht.IpfsDHT

	if *dest == "" {
		dht, err = NewKDHT(ctx, host, []multiaddr.Multiaddr{})
	} else {
		maddr, err := multiaddr.NewMultiaddr(*dest)

		if err != nil {
			log.WithField("error", err).Errorln("Convert addr to multiaddr failed.")
			return
		}

		dht, err = NewKDHT(ctx, host, []multiaddr.Multiaddr{maddr})
	}

	go node.Discover(ctx, host, dht, "test")

	select {}
}
