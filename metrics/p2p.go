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
	sendQueueCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "p2p_send_queue_count",
		Help: "P2P message send queue count.",
	})
)

func SendQueueCountInc() {
	sendQueueCount.Inc()
}

func SendQueueCountDec() {
	sendQueueCount.Dec()
}
