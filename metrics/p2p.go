/**
  @author: decision
  @date: 2023/7/3
  @note:
**/

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	sendQueueCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "p2p_send_queue_count",
		Help: "P2P message send queue count.",
	})
	recvQueueCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "p2p_recv_queue_count",
		Help: "P2P message receive queue count.",
	})
	gossipReceiveCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gossip_receive_counter",
		Help: "P2P network broadcast topic receive counter.",
	})
	gossipBlockReceiveCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gossip_block_receive_counter",
		Help: "P2P network broadcast topic receive blocks counter.",
	})
)

func SendQueueCountInc() {
	sendQueueCounter.Inc()
}

func RecvQueueCountInc() {
	recvQueueCounter.Inc()
}

func GossipReceiveCountInc() {
	gossipReceiveCounter.Inc()
}

func GossipReceiveBlocksCountInc() {
	gossipBlockReceiveCounter.Inc()
}

//
//func SendQueueCountDec() {
//	sendQueueCount.Dec()
//}
