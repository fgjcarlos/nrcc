package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FlowSummary is the compact per-tab representation consumed by the UI.
// Node-RED stores tabs and their nodes in one flat flows.json array; exposing
// that raw array made the frontend contract impossible to satisfy.
type FlowSummary struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Nodes       int    `json:"nodes"`
	Connections int    `json:"connections"`
	Disabled    bool   `json:"disabled"`
}

// FlowList is the response for GET /api/flows.
type FlowList struct {
	Available bool          `json:"available"`
	Flows     []FlowSummary `json:"flows"`
}

// FlowDetail groups the nodes belonging to one Node-RED tab.
type FlowDetail struct {
	ID    string                   `json:"id"`
	Label string                   `json:"label"`
	Nodes []map[string]interface{} `json:"nodes"`
}

// FlowMetrics contains structural metrics for one tab.
type FlowMetrics struct {
	NodeCount       int            `json:"nodeCount"`
	ConnectionCount int            `json:"connectionCount"`
	EntryPoints     []string       `json:"entryPoints"`
	ExitPoints      []string       `json:"exitPoints"`
	NodeTypes       map[string]int `json:"nodeTypes"`
	DisabledNodes   int            `json:"disabledNodes"`
}

// FlowService handles flow operations
type FlowService struct {
	dataDir string
}

// NewFlowService creates a new flow service
func NewFlowService(dataDir string) *FlowService {
	return &FlowService{
		dataDir: dataDir,
	}
}

// readRawFlows returns Node-RED's flat flows.json representation.
func (s *FlowService) readRawFlows() ([]map[string]interface{}, error) {
	flowsPath := filepath.Join(s.dataDir, "flows.json")

	// #nosec G304 -- flowsPath is built from operator-supplied dataDir + a constant filename; not request-derived.
	data, err := os.ReadFile(flowsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to read flows: %w", err)
	}

	var flows []map[string]interface{}
	if err := json.Unmarshal(data, &flows); err != nil {
		return nil, fmt.Errorf("failed to parse flows: %w", err)
	}

	return flows, nil
}

// GetFlows converts Node-RED's flat document into one summary per tab.
func (s *FlowService) GetFlows() (FlowList, error) {
	raw, err := s.readRawFlows()
	if err != nil {
		return FlowList{}, err
	}

	summaries := make([]FlowSummary, 0)
	for _, candidate := range raw {
		if stringField(candidate, "type") != "tab" {
			continue
		}
		id := stringField(candidate, "id")
		if id == "" {
			continue
		}
		detail := buildFlowDetail(candidate, raw)
		metrics := calculateFlowMetrics(detail.Nodes)
		summaries = append(summaries, FlowSummary{
			ID:          id,
			Label:       detail.Label,
			Nodes:       metrics.NodeCount,
			Connections: metrics.ConnectionCount,
			Disabled:    boolField(candidate, "disabled"),
		})
	}

	return FlowList{Available: true, Flows: summaries}, nil
}

// GetFlow returns a tab plus every node whose z property references it.
func (s *FlowService) GetFlow(id string) (FlowDetail, error) {
	raw, err := s.readRawFlows()
	if err != nil {
		return FlowDetail{}, err
	}
	for _, candidate := range raw {
		if stringField(candidate, "type") == "tab" && stringField(candidate, "id") == id {
			return buildFlowDetail(candidate, raw), nil
		}
	}
	return FlowDetail{}, fmt.Errorf("flow not found: %s", id)
}

// GetFlowMetrics calculates structural metrics from the nodes in one tab.
func (s *FlowService) GetFlowMetrics(id string) (FlowMetrics, error) {
	flow, err := s.GetFlow(id)
	if err != nil {
		return FlowMetrics{}, err
	}
	return calculateFlowMetrics(flow.Nodes), nil
}

// ExportFlows returns raw flows.json bytes for download
func (s *FlowService) ExportFlows() ([]byte, error) {
	flowsPath := filepath.Join(s.dataDir, "flows.json")

	// #nosec G304 -- flowsPath is built from operator-supplied dataDir + a constant filename; not request-derived.
	data, err := os.ReadFile(flowsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []byte("[]"), nil
		}
		return nil, fmt.Errorf("failed to export flows: %w", err)
	}

	return data, nil
}

func buildFlowDetail(tab map[string]interface{}, raw []map[string]interface{}) FlowDetail {
	id := stringField(tab, "id")
	label := strings.TrimSpace(stringField(tab, "label"))
	if label == "" {
		label = strings.TrimSpace(stringField(tab, "name"))
	}
	if label == "" {
		label = id
	}
	nodes := make([]map[string]interface{}, 0)
	for _, candidate := range raw {
		if stringField(candidate, "z") == id {
			nodes = append(nodes, candidate)
		}
	}
	return FlowDetail{ID: id, Label: label, Nodes: nodes}
}

func calculateFlowMetrics(nodes []map[string]interface{}) FlowMetrics {
	metrics := FlowMetrics{
		EntryPoints: make([]string, 0),
		ExitPoints:  make([]string, 0),
		NodeTypes:   make(map[string]int),
	}
	incoming := make(map[string]int, len(nodes))
	for _, node := range nodes {
		id := stringField(node, "id")
		if id != "" {
			incoming[id] = 0
		}
	}
	for _, node := range nodes {
		metrics.NodeCount++
		id := stringField(node, "id")
		typeName := stringField(node, "type")
		if typeName != "" {
			metrics.NodeTypes[typeName]++
		}
		if boolField(node, "disabled") || boolField(node, "d") {
			metrics.DisabledNodes++
		}
		connections := wireTargets(node["wires"])
		metrics.ConnectionCount += len(connections)
		if len(connections) == 0 && id != "" {
			metrics.ExitPoints = append(metrics.ExitPoints, id)
		}
		for _, target := range connections {
			if _, exists := incoming[target]; exists {
				incoming[target]++
			}
		}
	}
	for _, node := range nodes {
		id := stringField(node, "id")
		if id != "" && incoming[id] == 0 {
			metrics.EntryPoints = append(metrics.EntryPoints, id)
		}
	}
	return metrics
}

func wireTargets(value interface{}) []string {
	targets := make([]string, 0)
	outputs, ok := value.([]interface{})
	if !ok {
		return targets
	}
	for _, output := range outputs {
		wires, ok := output.([]interface{})
		if !ok {
			continue
		}
		for _, wire := range wires {
			if target, ok := wire.(string); ok && target != "" {
				targets = append(targets, target)
			}
		}
	}
	return targets
}

func stringField(value map[string]interface{}, key string) string {
	result, _ := value[key].(string)
	return result
}

func boolField(value map[string]interface{}, key string) bool {
	result, _ := value[key].(bool)
	return result
}
