package p2p

import (
	"context"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	log "github.com/sirupsen/logrus"
	"go-chronos/node"
	"time"
)

// Discover 基于 kademlia 协议发现其他节点
func Discover(ctx context.Context, h host.Host, dht *dht.IpfsDHT, rendezvous string) {
	var routingDiscovery = routing.NewRoutingDiscovery(dht)
	ttl, err := routingDiscovery.Advertise(ctx, rendezvous)

	if err != nil {
		log.WithField("error", err).Errorln("Routing discovery start failed.")
	}

	log.WithFields(log.Fields{
		"ttl": ttl,
	}).Infoln("Routing discovery start.")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	handler := node.GetHandlerInst()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dht.RefreshRoutingTable()
			peers, err := routingDiscovery.FindPeers(ctx, rendezvous, discovery.Limit(20))

			if err != nil {
				log.WithField("error", err).Errorln("Find peers failed.")
				continue
			}

			for p := range peers {
				log.Infoln(p.ID)

				if p.ID == h.ID() {
					continue
				}
				if h.Network().Connectedness(p.ID) != network.Connected {
					_, err := h.Network().DialPeer(ctx, p.ID)
					if err != nil {
						log.WithFields(log.Fields{
							"peerID": p.ID,
							"error":  err,
						}).Errorln("Connect to node failed.")
						continue
					}

					s, err := h.NewStream(ctx, p.ID, ProtocolId)
					_, err = handler.NewPeer(p.ID, &s)
					if err != nil {
						log.WithField("error", err).Errorln("Create new peer failed.")
						continue
					}
				}
			}
		}
	}

}
