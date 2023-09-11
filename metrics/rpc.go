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
	submitTransactionCountsMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rpc_submit_transaction_count",
		Help: "RPC interface receive submit transaction request counts.",
	})
)

func SubmitTxCountsMetricsInc() {
	submitTransactionCountsMetric.Inc()
}
