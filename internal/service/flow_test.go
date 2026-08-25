package service

import (
	"os"
	"path/filepath"
	"testing"
)

const testFlowsJSON = `[
  {"id":"tab-a","type":"tab","label":"Production","disabled":false},
  {"id":"inject-a","type":"inject","z":"tab-a","wires":[["function-a"]]},
  {"id":"function-a","type":"function","z":"tab-a","wires":[["debug-a"]]},
  {"id":"debug-a","type":"debug","z":"tab-a","d":true,"wires":[]},
  {"id":"tab-b","type":"tab","label":"Empty","disabled":true},
  {"id":"global-config","type":"global-config"}
]`

func writeTestFlows(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(testFlowsJSON), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestFlowServiceBuildsTabSummariesFromFlatNodeRedDocument(t *testing.T) {
	svc := NewFlowService(writeTestFlows(t))
	result, err := svc.GetFlows()
	if err != nil {
		t.Fatal(err)
	}
	if !result.Available || len(result.Flows) != 2 {
		t.Fatalf("result=%+v, want two available flows", result)
	}
	production := result.Flows[0]
	if production.ID != "tab-a" || production.Label != "Production" || production.Nodes != 3 || production.Connections != 2 || production.Disabled {
		t.Fatalf("production summary=%+v", production)
	}
	if !result.Flows[1].Disabled || result.Flows[1].Nodes != 0 {
		t.Fatalf("empty summary=%+v", result.Flows[1])
	}
}

func TestFlowServiceReturnsDetailAndMetricsForTab(t *testing.T) {
	svc := NewFlowService(writeTestFlows(t))
	detail, err := svc.GetFlow("tab-a")
	if err != nil {
		t.Fatal(err)
	}
	if detail.ID != "tab-a" || detail.Label != "Production" || len(detail.Nodes) != 3 {
		t.Fatalf("detail=%+v", detail)
	}
	metrics, err := svc.GetFlowMetrics("tab-a")
	if err != nil {
		t.Fatal(err)
	}
	if metrics.NodeCount != 3 || metrics.ConnectionCount != 2 || metrics.DisabledNodes != 1 {
		t.Fatalf("metrics=%+v", metrics)
	}
	if len(metrics.EntryPoints) != 1 || metrics.EntryPoints[0] != "inject-a" {
		t.Fatalf("entry points=%v", metrics.EntryPoints)
	}
	if len(metrics.ExitPoints) != 1 || metrics.ExitPoints[0] != "debug-a" {
		t.Fatalf("exit points=%v", metrics.ExitPoints)
	}
}

func TestFlowServiceMissingFileReturnsAvailableEmptyList(t *testing.T) {
	result, err := NewFlowService(t.TempDir()).GetFlows()
	if err != nil {
		t.Fatal(err)
	}
	if !result.Available || len(result.Flows) != 0 {
		t.Fatalf("result=%+v", result)
	}
}
