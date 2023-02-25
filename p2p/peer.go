package p2p

import (
	"github.com/libp2p/go-libp2p/core/peer"
)

type Peer struct {
	peerID peer.ID
}

func NewPeer(id peer.ID) (Peer, error) {

}
