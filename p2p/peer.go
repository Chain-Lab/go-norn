package p2p

import (
	"bufio"
	"context"
	"encoding/base64"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
	karmem "karmem.org/golang"
	"sync"
	"time"
)

const (
	pingInterval = 15 * time.Second
	//bufferSize   = 50 * 1024 * 1024
)

var (
	contextOnce sync.Once
	peerContext context.Context
	//writerPool  = sync.Pool{New: func() any { return karmem.NewWriter(1024) }}
)

type Peer struct {
	peerID peer.ID

	rw        *bufio.ReadWriter
	wg        sync.WaitGroup
	msgQueue  chan *Message
	sendQueue chan *Message

	wLock sync.RWMutex
	rLock sync.RWMutex
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

func NewPeer(id peer.ID, s *network.Stream, msgQueue chan *Message) (*Peer, error) {
	p := Peer{
		peerID: id,
		//rw:       bufio.NewReadWriter(bufio.NewReaderSize(*s, bufferSize), bufio.NewWriterSize(*s, bufferSize)),
		rw:        bufio.NewReadWriter(bufio.NewReader(*s), bufio.NewWriter(*s)),
		msgQueue:  msgQueue,
		sendQueue: make(chan *Message),
	}

	go p.Run()

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
	go p.writeLoop()
	return
}

func (p *Peer) Id() peer.ID {
	return p.peerID
}

func (p *Peer) pingLoop() {
	ping := time.NewTicker(pingInterval)
	defer p.wg.Done()

	log.Infoln("Start ping loop.")

	for {
		select {
		case <-ping.C:
			log.Debugln("Send ping to peer.")
			p.Send(StatusCodePingMsg, make([]byte, 0))
		}
	}
}

func (p *Peer) readLoop(errc chan<- error) {
	log.Traceln("Start read loop.")

	//var messagePool = sync.Pool{New: func() any { return new(Message) }}
	defer p.wg.Done()
	for {
		log.Traceln("New read loop.")
		dataBytes, err := p.rw.ReadBytes(0xff)

		//log.Infof("Read byte code data: %v", dataBytes)

		if err != nil {
			log.WithField("error", err).Errorln("Read bytes error.")
			errc <- err
			return
		}

		//log.WithField("length", len(dataBytes)).Infoln("Receive data bytes.")

		if len(dataBytes) == 0 {
			continue
		}
		//log.Infof("Receive byte data %v", dataBytes)
		dataBytes = dataBytes[:len(dataBytes)-1]

		decodedPayload := make([]byte, base64.StdEncoding.DecodedLen(len(dataBytes)))
		l, _ := base64.StdEncoding.Decode(decodedPayload, dataBytes)

		msg := new(Message)
		msg.ReadAsRoot(karmem.NewReader(decodedPayload[:l]))

		now := time.Now()

		// todo: 修改编码为int64
		msg.ReceiveAt = now.UnixMilli()

		//log.WithFields(log.Fields{
		//	"code":   msg.Code,
		//	"length": len(dataBytes),
		//	"data":   hex.EncodeToString(decodedPayload),
		//}).Infoln("Receive message.")
		p.handle(msg)

		//msg.Reset()
		//messagePool.Put(msg)
	}
}

func (p *Peer) handle(msg *Message) {
	switch {
	case msg.Code == StatusCodePingMsg:
		log.WithField("peer", p.peerID).Traceln("Receive peer ping message.")
		p.Send(StatusCodePongMsg, make([]byte, 0))
		return
		// todo: 怎么将消息传入到上层进行处理
		// todo: channel 发送消息到 node/peer 中处理
	case msg.Code == StatusCodePongMsg:
		return
	default:
		p.msgQueue <- msg
	}
	return
}

func (p *Peer) disconnect() {

}

// Send 方法用于提供一个通用的消息发送接口
func (p *Peer) Send(msgCode StatusCode, payload []byte) {
	msg := Message{
		Code:      msgCode,
		Size:      uint32(len(payload)),
		Payload:   payload,
		ReceiveAt: 0,
	}

	//log.WithField("payload", hex.EncodeToString(payload)).Infoln("Send msg to channel.")
	p.sendQueue <- &msg
}

func (p *Peer) writeLoop() {
	log.Infoln("Start write loop.")
	for {
		select {
		case msg := <-p.sendQueue:
			//msgWriter := writerPool.Get().(*karmem.Writer)
			msgWriter := karmem.NewWriter(1024)
			//log.WithFields(log.Fields{
			//	"code":    msg.Code,
			//	"payload": hex.EncodeToString(msg.Payload),
			//}).Debugln("Start write buffer.")

			if _, err := msg.WriteAsRoot(msgWriter); err != nil {
				log.WithField("error", err).Errorln("Encode data failed.")
				break
			}

			msgBytes := msgWriter.Bytes()

			encodedPayload := make([]byte, base64.StdEncoding.EncodedLen(len(msgBytes)))
			base64.StdEncoding.Encode(encodedPayload, msgBytes)
			encodedPayload = append(encodedPayload, 0xff)

			length, err := p.rw.Write(encodedPayload)

			if err != nil {
				log.WithFields(
					log.Fields{
						"error":  err,
						"length": length,
						"code":   msg.Code,
					}).Errorln("Send data to peer errored.")
				//log.Debugf("Data bytes: %v", msgBytes)
				break
			}

			// 这里必须强制 Flush， 否则短消息收不到
			p.rw.Flush()

			//log.WithFields(log.Fields{
			//	"length":    length,
			//	"code":      msg.Code,
			//	"data":      hex.EncodeToString(msgBytes),
			//	"timestamp": time.Now().Nanosecond(),
			//}).Infoln("Send message to peer.")

			//msgWriter.Reset()
			//writerPool.Put(msgWriter)
		}
	}
}
