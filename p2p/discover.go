package p2p

import (
	"context"
	"github.com/libp2p/go-libp2p-kad-dht"
	host "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	log "github.com/sirupsen/logrus"
	"time"
)

// Discover 基于 kademlia 协议发现其他节点
func Discover(ctx context.Context, h host.Host, dht *dht.IpfsDHT, rendezvous string) {
	var routingDiscovery = routing.NewRoutingDiscovery(dht)
	ttl, err := routingDiscovery.Advertise(ctx, rendezvous)

	if err != nil {
		log.Errorln("Routing discovery start failed.")
	}

	log.WithFields(log.Fields{
		"ttl": ttl,
	}).Infoln("Routing discovery start.")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			peers, err := routingDiscovery.FindPeers(ctx, rendezvous)

			if err != nil {
				log.WithField("error", err).Debugln("Find peers failed.")
				continue
			}

			for p := range peers {
				if p.ID == h.ID() {
					continue
				}
				if h.Network().Connectedness(p.ID) != network.Connected {
					conn, err := h.Network().DialPeer(ctx, p.ID)
					if err != nil {
						log.WithField("peerID", p.ID).Debugln("Connect to node failed.")
						continue
					}
				}
			}
		}
	}

}
