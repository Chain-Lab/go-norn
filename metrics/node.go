/**
  @author: decision
  @date: 2023/6/30
  @note:
**/

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"strconv"
)

var (
	// 时间同步的差值
	timeSyncDelta = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "node_time_sync_delta",
		Help: "Time sync delta per round sync.",
	})
	// 插入数据库的交易数量
	transactionInsertCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "node_transaction_insert_count",
		Help: "Node transaction insert count.",
	})
	// 连接的对端节点数量
	connectNodeCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "node_connect_count",
		Help: "The count of connected node.",
	})
	// 区块高度数量
	blockHeightCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "block_height_count",
		Help: "Block height count",
	})
	// 创建的协程相关的代码
	RoutineCreateCodeCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "routine_create_summary",
		Help: "The summary for create new routine.",
	},
		[]string{"code"},
	)
	// 从对端接收到的消息类型
	HandleReceivedMessageCodeCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "handle_received_message_codes",
			Help: "The counter that record received message code",
		},
		[]string{"code"},
	)
	// 时间同步状态
	TimeSyncrStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "time_syncer_status",
		Help: "The status in time syncer.",
	})
	// 区块同步状态
	BlockSyncrStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "block_syncer_status",
		Help: "The status in time syncer.",
	})
	transactionInsertCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "transaction_insert_counter",
		Help: "The counter for inserted txs.",
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

func RoutineCreateCounterObserve(value int) {
	RoutineCreateCodeCounter.WithLabelValues(strconv.Itoa(value)).Inc()
}

func BlockHeightSet(height int64) {
	blockHeightCount.Set(float64(height))
}

func RecordHandleReceivedCode(code int) {
	HandleReceivedMessageCodeCounter.WithLabelValues(strconv.Itoa(code)).Inc()
}

func TimeSyncerStatusSet(code int8) {
	TimeSyncrStatus.Set(float64(code))
}

func BlockSyncerStatusSet(code int8) {
	BlockSyncrStatus.Set(float64(code))
}

func TransactionInsertInc() {
	transactionInsertCounter.Inc()
}
