/**
  @author: decision
  @date: 2023/6/30
  @note:
**/

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	timeSyncDelta = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "node_time_sync_delta",
		Help: "Time sync delta per round sync.",
	})
	transactionInsertCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "node_transaction_insert_count",
		Help: "Node transaction insert count.",
	})
)

func TimeSyncDeltaSet(value float64) {
	timeSyncDelta.Set(value)
}

func TransactionInsertAdd(value float64) {
	transactionInsertCount.Add(value)
}
