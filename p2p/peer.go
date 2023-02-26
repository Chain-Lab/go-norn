package p2p

import (
	"bufio"
	"context"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
)

type Peer struct {
	peerID peer.ID
	conn   network.Conn

	rw *bufio.ReadWriter
}

func NewPeer(ctx context.Context, id peer.ID, conn *network.Conn) (*Peer, error) {
	s, err := (*conn).NewStream(ctx)

	if err != nil {
		log.WithField("error", err).Debugln("Create new peer failed.")
		return nil, err
	}

	p := Peer{
		peerID: id,
		conn:   *conn,
		rw:     bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s)),
	}

	return &p, nil
}

func (p *Peer) Id() peer.ID {
	return p.peerID
}

func (p *Peer) RemoteId() peer.ID {
	return p.conn.RemotePeer()
}

func (p *Peer) readLoop(errc chan<- error) {
	for {
		msg, err := p.rw.Read
	}
}
