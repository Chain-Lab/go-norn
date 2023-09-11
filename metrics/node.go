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
	connectNodeCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "node_connect_count",
		Help: "The count of connected node.",
	})
	RoutineCreateHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "routine_create_summary",
		Help: "The summary for create new routine.",
		Buckets: []float64{
			0.0, 1.0, 2.0, 3.0, 4.0,
			5.0, 6.0, 7.0, 8.0, 9.0, 10.0,
			11.0, 12.0, 13.0, 14.0, 15.0,
			16.0, 17.0, 18.0, 19.0, 20.0,
			21.0, 22.0, 23.0, 24.0, 25.0,
			26.0, 27.0, 28.0, 29.0, 30.0},
	})
)

func TimeSyncDeltaSet(value float64) {
	timeSyncDelta.Set(value)
}

func TransactionInsertAdd(value float64) {
	transactionInsertCount.Add(value)
}

func ConnectedNodeInc() {
	connectNodeCount.Inc()
}

func ConnectedNodeDec() {
	connectNodeCount.Dec()
}

func RoutineCreateHistogramObserve(value float64) {
	RoutineCreateHistogram.Observe(value)
}
