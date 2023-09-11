/**
  @author: decision
  @date: 2023/7/3
  @note:
**/

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shirou/gopsutil/cpu"
	"time"
)

var (
	cpuPercentGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "regular_cpu_usage",
		Help: "Regular info for cpu usage.",
	})
)

func getCpuPercent() float64 {
	percent, _ := cpu.Percent(time.Second, false)
	return percent[0]
}

func RegularMetricsRoutine() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		// 每 1s 通过 metrics 记录 cpu 占用信息
		case <-ticker.C:
			cpuPercentGauge.Set(getCpuPercent())
		}
	}
}
