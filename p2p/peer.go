package p2p

import (
	"bufio"
	"context"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
	karmem "karmem.org/golang"
	"sync"
	"time"
)

const (
	pingInterval = 3 * time.Second
)

var (
	contextOnce sync.Once
	peerContext context.Context
)

type Peer struct {
	peerID peer.ID

	rw *bufio.ReadWriter
	wg sync.WaitGroup
}

func GetPeerContext() context.Context {
	contextOnce.Do(func() {
		peerContext = context.Background()
	})

	return peerContext
}

//func newPeer() *Peer {
//	p := Peer{
//		wg:
//	}
//}

func NewPeer(id peer.ID, s *network.Stream) (*Peer, error) {
	p := Peer{
		peerID: id,
		rw:     bufio.NewReadWriter(bufio.NewReader(*s), bufio.NewWriter(*s)),
	}

	return &p, nil
}

func (p *Peer) Run() {
	var (
		readErr = make(chan error, 1)
	)

	log.WithField("peer", p.peerID).Infoln("Start run peer instance.")

	p.wg.Add(2)
	go p.pingLoop()
	go p.readLoop(readErr)
	return
}

func (p *Peer) Id() peer.ID {
	return p.peerID
}

func (p *Peer) pingLoop() {
	ping := time.NewTimer(pingInterval)
	defer p.wg.Done()
	defer ping.Stop()

	log.Infoln("Start ping loop.")

	for {
		select {
		case <-ping.C:
			log.Debugln("Send ping to peer.")
			Send(p.rw, StatusCodePingMsg, *karmem.NewWriter(0))
			ping.Reset(pingInterval)
		}
	}
}

func (p *Peer) readLoop(errc chan<- error) {
	log.Traceln("Start read loop.")

	var messagePool = sync.Pool{New: func() any { return new(Message) }}
	defer p.wg.Done()
	for {
		log.Traceln("New read loop.")
		dataBytes, err := p.rw.ReadBytes(0xff)

		if err != nil {
			log.WithField("error", err).Errorln("Read bytes error.")
			errc <- err
			return
		}

		//log.WithField("length", len(dataBytes)).Infoln("Receive data bytes.")

		if len(dataBytes) == 0 {
			continue
		}
		dataBytes = dataBytes[:len(dataBytes)-1]

		msg := messagePool.Get().(*Message)
		msg.ReadAsRoot(karmem.NewReader(dataBytes))

		now := time.Now()

		// todo: 修改编码为int64
		msg.ReceiveAt = uint32(now.UnixMilli())

		if err = p.handle(msg); err != nil {
			errc <- err
			return
		}
	}
}

func (p *Peer) handle(msg *Message) error {
	switch {
	case msg.Code == StatusCodePingMsg:
		karmemWriter := karmem.NewWriter(0)
		log.WithField("peer", p.peerID).Infoln("Receive peer ping message.")
		go Send(p.rw, StatusCodePongMsg, *karmemWriter)
	}
	return nil
}

func (p *Peer) disconnect() {

}
