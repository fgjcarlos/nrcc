package metrics

import (
	"strings"
	"testing"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// stubProcessManagerSource is a test double that implements ProcessManagerSource.
type stubProcessManagerSource struct {
	status model.RuntimeStatus
}

func (s *stubProcessManagerSource) Status() model.RuntimeStatus {
	return s.status
}

// TestProcessCollector_Describe_EmitsThreeDescriptors verifies that Describe sends exactly
// three descriptors to the channel: running, restarts_total, and uptime_seconds.
func TestProcessCollector_Describe_EmitsThreeDescriptors(t *testing.T) {
	pc := newProcessCollector()

	ch := make(chan *prometheus.Desc, 10)
	pc.Describe(ch)
	close(ch)

	var descs []*prometheus.Desc
	for d := range ch {
		descs = append(descs, d)
	}

	if len(descs) != 3 {
		t.Errorf("Describe() sent %d descriptors, want 3", len(descs))
	}
}

// TestProcessCollector_Collect_NilSource_YieldsZeroGauges verifies that when no source is
// wired, all three gauges are emitted with value 0.
func TestProcessCollector_Collect_NilSource_YieldsZeroGauges(t *testing.T) {
	pc := newProcessCollector()

	reg := prometheus.NewRegistry()
	reg.MustRegister(pc)

	want := `
# HELP nrcc_nodered_restarts_total Point-in-time restart count for the Node-RED process as reported by the process manager.
# TYPE nrcc_nodered_restarts_total gauge
nrcc_nodered_restarts_total 0
# HELP nrcc_nodered_running 1 if Node-RED is currently running, 0 otherwise.
# TYPE nrcc_nodered_running gauge
nrcc_nodered_running 0
# HELP nrcc_nodered_uptime_seconds Number of seconds Node-RED has been running since last start.
# TYPE nrcc_nodered_uptime_seconds gauge
nrcc_nodered_uptime_seconds 0
`
	if err := testutil.GatherAndCompare(reg, strings.NewReader(want),
		"nrcc_nodered_running",
		"nrcc_nodered_restarts_total",
		"nrcc_nodered_uptime_seconds",
	); err != nil {
		t.Errorf("unexpected metrics with nil source: %v", err)
	}
}

// TestProcessCollector_Collect_RunningProcess_YieldsCorrectGauges verifies that when a
// running process is reported, the gauges reflect the correct values.
func TestProcessCollector_Collect_RunningProcess_YieldsCorrectGauges(t *testing.T) {
	pc := newProcessCollector()
	pc.setSource(&stubProcessManagerSource{
		status: model.RuntimeStatus{
			Status:       "running",
			Uptime:       42,
			RestartCount: 3,
		},
	})

	reg := prometheus.NewRegistry()
	reg.MustRegister(pc)

	want := `
# HELP nrcc_nodered_restarts_total Point-in-time restart count for the Node-RED process as reported by the process manager.
# TYPE nrcc_nodered_restarts_total gauge
nrcc_nodered_restarts_total 3
# HELP nrcc_nodered_running 1 if Node-RED is currently running, 0 otherwise.
# TYPE nrcc_nodered_running gauge
nrcc_nodered_running 1
# HELP nrcc_nodered_uptime_seconds Number of seconds Node-RED has been running since last start.
# TYPE nrcc_nodered_uptime_seconds gauge
nrcc_nodered_uptime_seconds 42
`
	if err := testutil.GatherAndCompare(reg, strings.NewReader(want),
		"nrcc_nodered_running",
		"nrcc_nodered_restarts_total",
		"nrcc_nodered_uptime_seconds",
	); err != nil {
		t.Errorf("unexpected metrics for running process: %v", err)
	}
}

// TestProcessCollector_Collect_StoppedProcess_YieldsZeroRunning verifies that when the
// process status is "stopped", nrcc_nodered_running is 0 but uptime is reported.
func TestProcessCollector_Collect_StoppedProcess_YieldsZeroRunning(t *testing.T) {
	pc := newProcessCollector()
	pc.setSource(&stubProcessManagerSource{
		status: model.RuntimeStatus{
			Status: "stopped",
			Uptime: 0,
		},
	})

	reg := prometheus.NewRegistry()
	reg.MustRegister(pc)

	want := `
# HELP nrcc_nodered_restarts_total Point-in-time restart count for the Node-RED process as reported by the process manager.
# TYPE nrcc_nodered_restarts_total gauge
nrcc_nodered_restarts_total 0
# HELP nrcc_nodered_running 1 if Node-RED is currently running, 0 otherwise.
# TYPE nrcc_nodered_running gauge
nrcc_nodered_running 0
# HELP nrcc_nodered_uptime_seconds Number of seconds Node-RED has been running since last start.
# TYPE nrcc_nodered_uptime_seconds gauge
nrcc_nodered_uptime_seconds 0
`
	if err := testutil.GatherAndCompare(reg, strings.NewReader(want),
		"nrcc_nodered_running",
		"nrcc_nodered_restarts_total",
		"nrcc_nodered_uptime_seconds",
	); err != nil {
		t.Errorf("unexpected metrics for stopped process: %v", err)
	}
}
