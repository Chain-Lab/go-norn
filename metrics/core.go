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
	poolTransactionsMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "core_tx_pool_transactions",
		Help: "Transaction count in memory pool.",
	})
	packageBlockMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "core_package_block_time",
		Help: "Package block time usage.",
	})
	verifyTransactionMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "core_transaction_verify_time",
		Help: "Transaction verify time usage.",
	})
	blockBufferCountMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "core_block_buffer_count",
		Help: "Block buffer count",
	})
)

func TxPoolMetricsInc() {
	poolTransactionsMetric.Inc()
}

func TxPoolMetricsDec() {
	poolTransactionsMetric.Dec()
}

func PackageBlockMetricsSet(usage float64) {
	packageBlockMetric.Set(usage)
}

func VerifyTransactionMetricsSet(usage float64) {
	verifyTransactionMetric.Set(usage)
}

func BlockBufferCountMetricsDec(value int) {
	blockBufferCountMetric.Sub((float64(value)))
}

func BlockBufferCountMetricsInc() {
	blockBufferCountMetric.Inc()
}
