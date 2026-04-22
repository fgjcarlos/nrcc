package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

type stubFlowAnalysisProvider struct {
	metadata model.FlowAnalysisProvider
	result   FlowAnalysisPayload
	err      error
	prompt   string
}

func (s *stubFlowAnalysisProvider) Metadata() model.FlowAnalysisProvider {
	return s.metadata
}

func (s *stubFlowAnalysisProvider) Analyze(_ context.Context, prompt string) (FlowAnalysisPayload, error) {
	s.prompt = prompt
	if s.err != nil {
		return FlowAnalysisPayload{}, s.err
	}
	return s.result, nil
}

func TestFlowServiceListAndDetail(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	userDir := filepath.Join(dataDir, "nodered")
	if err := platform.WriteFileAtomic(filepath.Join(userDir, "flows.json"), []byte(`[
		{"id":"tab-a","type":"tab","label":"Main Flow"},
		{"id":"tab-b","type":"tab","label":"Secondary"},
		{"id":"inject-1","type":"inject","z":"tab-a","name":"Start","wires":[["debug-1","custom-1"]]},
		{"id":"debug-1","type":"debug","z":"tab-a","name":"Logger","wires":[]},
		{"id":"custom-1","type":"acme-widget","z":"tab-a","name":"Widget","d":true,"wires":[["subflow-use-1"]]},
		{"id":"subflow-use-1","type":"subflow:sub-1","z":"tab-a","name":"Child flow","wires":[["http-1"]]},
		{"id":"group-1","type":"group","z":"tab-a","name":"Ignore Group"},
		{"id":"http-1","type":"http in","z":"tab-b","name":"Inbound","wires":[["debug-2"]]},
		{"id":"debug-2","type":"debug","z":"tab-b","name":"Done","wires":[]},
		{"id":"cfg-1","type":"mqtt-broker","name":"Broker"}
	]`), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic(flows.json) error = %v", err)
	}

	service := NewFlowService(dataDir)

	list, err := service.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if list.Source.Path != filepath.Join(userDir, "flows.json") {
		t.Fatalf("List() source path = %q", list.Source.Path)
	}
	if !list.Source.ReadOnly {
		t.Fatal("List() source readOnly = false, want true")
	}
	if list.Summary.FlowCount != 2 || list.Summary.NodeCount != 6 {
		t.Fatalf("List() summary = %+v", list.Summary)
	}
	if len(list.Items) != 2 {
		t.Fatalf("List() items len = %d, want 2", len(list.Items))
	}

	main := list.Items[0]
	if main != (model.FlowSummary{
		ID:                "tab-a",
		Label:             "Main Flow",
		NodeCount:         4,
		DisabledNodeCount: 1,
		CustomNodeCount:   1,
		InboundWireCount:  3,
		OutboundWireCount: 4,
		SubflowUsageCount: 1,
	}) {
		t.Fatalf("List() main flow = %+v", main)
	}

	secondary := list.Items[1]
	if secondary.InboundWireCount != 2 || secondary.OutboundWireCount != 1 || secondary.NodeCount != 2 {
		t.Fatalf("List() secondary flow = %+v", secondary)
	}

	detail, err := service.Get("tab-a")
	if err != nil {
		t.Fatalf("Get(tab-a) error = %v", err)
	}
	if detail.Flow.Label != "Main Flow" || len(detail.Flow.Nodes) != 4 {
		t.Fatalf("Get(tab-a) flow = %+v", detail.Flow)
	}
	if detail.Flow.Nodes[0].Name != "Child flow" {
		t.Fatalf("Get(tab-a) first node = %+v", detail.Flow.Nodes[0])
	}
	if len(detail.Flow.NodeTypes) != 4 {
		t.Fatalf("Get(tab-a) node types = %+v", detail.Flow.NodeTypes)
	}
}

func TestFlowServiceUsesConfiguredUserDirAndFlowFile(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	customUserDir := filepath.Join(dataDir, "custom-user-dir")
	config := model.DefaultFullAppConfig()
	config.Flows.UserDir = customUserDir
	config.Flows.FlowFile = "custom-flows.json"
	if err := platform.WriteJSONAtomic(filepath.Join(dataDir, "config.json"), config); err != nil {
		t.Fatalf("WriteJSONAtomic(config.json) error = %v", err)
	}
	if err := platform.WriteFileAtomic(filepath.Join(customUserDir, "custom-flows.json"), []byte(`[
		{"id":"tab-a","type":"tab","label":"Configured"},
		{"id":"inject-1","type":"inject","z":"tab-a","name":"Start","wires":[]}
	]`), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic(custom-flows.json) error = %v", err)
	}

	list, err := NewFlowService(dataDir).List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if list.Source.UserDir != customUserDir || list.Source.FlowFile != "custom-flows.json" {
		t.Fatalf("List() source = %+v", list.Source)
	}
}

func TestFlowServiceGetNotFound(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	if err := platform.WriteFileAtomic(filepath.Join(dataDir, "nodered", "flows.json"), []byte(`[{"id":"tab-a","type":"tab","label":"Main"}]`), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic(flows.json) error = %v", err)
	}

	_, err := NewFlowService(dataDir).Get("missing")
	if !errors.Is(err, ErrFlowNotFound) {
		t.Fatalf("Get(missing) error = %v, want ErrFlowNotFound", err)
	}
}

func TestFlowServiceAnalyze(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	if err := platform.WriteFileAtomic(filepath.Join(dataDir, "nodered", "flows.json"), []byte(`[
		{"id":"tab-a","type":"tab","label":"Main"},
		{"id":"inject-1","type":"inject","z":"tab-a","name":"Start","wires":[["debug-1"]]},
		{"id":"debug-1","type":"debug","z":"tab-a","name":"Logger","wires":[]}
	]`), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic(flows.json) error = %v", err)
	}

	provider := &stubFlowAnalysisProvider{
		metadata: model.FlowAnalysisProvider{Name: "ollama", Model: "llama3.2", Local: true},
		result: FlowAnalysisPayload{
			Summary:     "Resumen del flujo.",
			Strengths:   []string{"- Entrada clara"},
			Issues:      []string{"* Falta manejo de errores"},
			Suggestions: []string{"Agregar observabilidad adicional"},
		},
	}

	analysis, err := NewFlowServiceWithProvider(dataDir, provider).Analyze(context.Background(), "tab-a")
	if err != nil {
		t.Fatalf("Analyze(tab-a) error = %v", err)
	}
	if analysis.Flow.ID != "tab-a" || analysis.Provider.Name != "ollama" {
		t.Fatalf("Analyze(tab-a) = %+v", analysis)
	}
	if got := strings.Join(analysis.Strengths, ","); got != "Entrada clara" {
		t.Fatalf("Analyze(tab-a) strengths = %q", got)
	}
	if !strings.Contains(provider.prompt, "Main") || !strings.Contains(provider.prompt, "inject:1") {
		t.Fatalf("Analyze(tab-a) prompt = %q", provider.prompt)
	}
	if !analysis.Advisory {
		t.Fatal("Analyze(tab-a) advisory = false, want true")
	}
}
