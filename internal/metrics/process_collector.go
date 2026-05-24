package metrics

import (
	"sync"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/prometheus/client_golang/prometheus"
)

// ProcessManagerSource is the interface that ProcessCollector uses to read
// live Node-RED runtime status. *service.ProcessManager satisfies this interface.
type ProcessManagerSource interface {
	Status() model.RuntimeStatus
}

// ProcessCollector is a prometheus.Collector that exports Node-RED process gauges.
// It is safe for concurrent use.
type ProcessCollector struct {
	mu     sync.Mutex
	source ProcessManagerSource

	descRunning  *prometheus.Desc
	descRestarts *prometheus.Desc
	descUptime   *prometheus.Desc
}

// newProcessCollector creates a ProcessCollector with descriptors but no source wired yet.
func newProcessCollector() *ProcessCollector {
	return &ProcessCollector{
		descRunning: prometheus.NewDesc(
			"nrcc_nodered_running",
			"1 if Node-RED is currently running, 0 otherwise.",
			nil, nil,
		),
		descRestarts: prometheus.NewDesc(
			"nrcc_nodered_restarts_total",
			"Point-in-time restart count for the Node-RED process as reported by the process manager.",
			nil, nil,
		),
		descUptime: prometheus.NewDesc(
			"nrcc_nodered_uptime_seconds",
			"Number of seconds Node-RED has been running since last start.",
			nil, nil,
		),
	}
}

// setSource wires a ProcessManagerSource. Thread-safe.
func (pc *ProcessCollector) setSource(src ProcessManagerSource) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.source = src
}

// Describe sends the three gauge descriptors to the channel.
func (pc *ProcessCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- pc.descRunning
	ch <- pc.descRestarts
	ch <- pc.descUptime
}

// Collect reads the current Node-RED status and sends gauge metrics.
// If no source is wired (nil), all gauges are emitted with value 0.
func (pc *ProcessCollector) Collect(ch chan<- prometheus.Metric) {
	pc.mu.Lock()
	src := pc.source
	pc.mu.Unlock()

	if src == nil {
		ch <- prometheus.MustNewConstMetric(pc.descRunning, prometheus.GaugeValue, 0)
		ch <- prometheus.MustNewConstMetric(pc.descRestarts, prometheus.GaugeValue, 0)
		ch <- prometheus.MustNewConstMetric(pc.descUptime, prometheus.GaugeValue, 0)
		return
	}

	status := src.Status()

	var running float64
	if status.Status == "running" {
		running = 1
	}

	ch <- prometheus.MustNewConstMetric(pc.descRunning, prometheus.GaugeValue, running)
	ch <- prometheus.MustNewConstMetric(pc.descRestarts, prometheus.GaugeValue, float64(status.RestartCount))
	ch <- prometheus.MustNewConstMetric(pc.descUptime, prometheus.GaugeValue, float64(status.Uptime))
}
