// Package metrics
// @Description: 主要是和交易、缓冲相关的指标信息
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 交易池交易数量指标
	poolTransactionsMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "core_tx_pool_transactions",
		Help: "Transaction count in memory pool.",
	})
	// 打包区块的耗时
	packageBlockMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "core_package_block_time",
		Help: "Package block time usage.",
	})
	// 验证交易所消耗的时间
	verifyTransactionMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "core_transaction_verify_time",
		Help: "Transaction verify time usage.",
	})
	// 第二缓冲的区块数量
	coreBufferSecondQueueMetrices = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "core_buffer_second_queue",
		Help: "Core buffer second blocks",
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

func SecondBufferInc() {
	coreBufferSecondQueueMetrices.Inc()
}

func SecondBufferDec() {
	coreBufferSecondQueueMetrices.Dec()
}
