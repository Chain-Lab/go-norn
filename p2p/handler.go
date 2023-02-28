package p2p

import (
	"github.com/libp2p/go-libp2p/core/network"
	log "github.com/sirupsen/logrus"
)

var (
	ProtocolId string = "/chronos/1.0.0"
)

// HandleStream 用于在收到对端连接时候处理 stream, 在这里构建 peer 用于通信
func HandleStream(s network.Stream) {
	//ctx := GetPeerContext()
	conn := s.Conn()
	//peer, err := NewPeer(ctx, conn.RemotePeer(), &conn)
	peer, err := NewPeer(conn.RemotePeer(), &s)

	log.Infoln("Receive new stream, handle stream.")

	if err != nil {
		log.WithField("error", err).Errorln("Handle stream error.")
		return
	}

	go peer.Run()
}
