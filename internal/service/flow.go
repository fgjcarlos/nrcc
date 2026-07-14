package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

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

// GetFlows returns all flows from flows.json
func (s *FlowService) GetFlows() ([]interface{}, error) {
	flowsPath := filepath.Join(s.dataDir, "flows.json")

	data, err := os.ReadFile(flowsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to read flows: %w", err)
	}

	var flows []interface{}
	if err := json.Unmarshal(data, &flows); err != nil {
		return nil, fmt.Errorf("failed to parse flows: %w", err)
	}

	return flows, nil
}

// GetFlow returns a single flow node by ID
func (s *FlowService) GetFlow(id string) (interface{}, error) {
	flows, err := s.GetFlows()
	if err != nil {
		return nil, err
	}

	for _, flow := range flows {
		flowMap, ok := flow.(map[string]interface{})
		if !ok {
			continue
		}

		if flowID, ok := flowMap["id"].(string); ok && flowID == id {
			return flow, nil
		}
	}

	return nil, fmt.Errorf("flow not found: %s", id)
}

// ExportFlows returns raw flows.json bytes for download
func (s *FlowService) ExportFlows() ([]byte, error) {
	flowsPath := filepath.Join(s.dataDir, "flows.json")

	data, err := os.ReadFile(flowsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []byte("[]"), nil
		}
		return nil, fmt.Errorf("failed to export flows: %w", err)
	}

	return data, nil
}

// FlowService methods (ExportFlows shown above) — Analyze was removed in #395.
