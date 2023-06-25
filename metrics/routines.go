/**
  @author: decision
  @date: 2023/7/7
  @note:
**/

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RespondTimeSyncRoutineGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "routines_respond_time_sync",
		Help: "Record routine counts in time sync.",
	})
	RespondGetSyncStatusGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "routines_respond_get_sync_status",
		Help: "Record routine counts in get sync status.",
	})
)
