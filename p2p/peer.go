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
	pingInterval = 15 * time.Second
)

var (
	contextOnce sync.Once
	peerContext context.Context
	writerPool  = sync.Pool{New: func() any { return karmem.NewWriter(1024) }}
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
			p.Send(StatusCodePingMsg, _Null)
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
		log.WithField("peer", p.peerID).Traceln("Receive peer ping message.")
		go p.Send(StatusCodePongMsg, _Null)
		// todo: 怎么将消息传入到上层进行处理
		// todo: channel 发送消息到 node/peer 中处理
	}
	return nil
}

func (p *Peer) disconnect() {

}

// Send 方法用于提供一个通用的消息发送接口
func (p *Peer) Send(msgcode StatusCode, payload []byte) {
	// todo: 这里的 ReadWriter 传值是否存在问题, 此外还需要传入空值发送 ping/pong 信息
	msgWriter := writerPool.Get().(*karmem.Writer)
	defer msgWriter.Reset()
	defer writerPool.Put(msgWriter)

	// todo: 这里数据的大小暂时留空，作为冗余字段
	// todo: 观察一下这里处理数据会不会有较高的耗时，特别是数据比较大的情况下
	msg := Message{
		Code:      msgcode,
		Size:      uint32(len(payload)),
		Payload:   payload,
		ReceiveAt: 0,
	}

	if _, err := msg.WriteAsRoot(msgWriter); err != nil {
		log.WithField("error", err).Errorln("Encode data failed.")
		return
	}

	msgBytes := append(msgWriter.Bytes(), 0xff)
	length, err := p.rw.Write(msgBytes)
	if err != nil {
		log.WithField("error", err).Errorln("Send data to peer errored.")
		return
	}
	p.rw.Flush()

	log.WithFields(log.Fields{
		"length": length,
	}).Debugln("Send message to peer.")

	msgWriter.Reset()
	writerPool.Put(msgWriter)

	//return nil
}
