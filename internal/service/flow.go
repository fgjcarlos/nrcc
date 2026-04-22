package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

var ErrFlowNotFound = errors.New("flow not found")

var builtInFlowNodeTypes = map[string]struct{}{
	"catch":         {},
	"change":        {},
	"comment":       {},
	"complete":      {},
	"csv":           {},
	"debug":         {},
	"delay":         {},
	"exec":          {},
	"file":          {},
	"file in":       {},
	"function":      {},
	"group":         {},
	"http in":       {},
	"http request":  {},
	"http response": {},
	"inject":        {},
	"join":          {},
	"json":          {},
	"link call":     {},
	"link in":       {},
	"link out":      {},
	"mqtt-broker":   {},
	"mqtt in":       {},
	"mqtt out":      {},
	"rbe":           {},
	"range":         {},
	"split":         {},
	"status":        {},
	"subflow":       {},
	"switch":        {},
	"tab":           {},
	"template":      {},
	"trigger":       {},
	"unknown":       {},
	"websocket in":  {},
	"websocket out": {},
	"xml":           {},
	"yaml":          {},
	"tls-config":    {},
	"tcp in":        {},
	"tcp out":       {},
	"tcp request":   {},
	"udp in":        {},
	"udp out":       {},
	"serial-port":   {},
	"serial in":     {},
	"serial out":    {},
	"watch":         {},
	"batch":         {},
	"sort":          {},
	"junction":      {},
	"dns lookup":    {},
	"e-mail":        {},
	"e-mail in":     {},
	"tail":          {},
	"ui_base":       {},
	"ui_group":      {},
	"ui_tab":        {},
}

type FlowService struct {
	dataDir       string
	configService ConfigService
	provider      FlowAnalysisProvider
}

type FlowAnalysisProvider interface {
	Metadata() model.FlowAnalysisProvider
	Analyze(ctx context.Context, prompt string) (FlowAnalysisPayload, error)
}

type FlowAnalysisPayload struct {
	Summary     string   `json:"summary"`
	Strengths   []string `json:"strengths"`
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
}

type FlowAnalysisProviderUnavailableError struct {
	Provider string
	Model    string
	Message  string
	Action   string
	Cause    error
}

func (e *FlowAnalysisProviderUnavailableError) Error() string {
	if e == nil || e.Message == "" {
		return "flow analysis provider unavailable"
	}
	return e.Message
}

func (e *FlowAnalysisProviderUnavailableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type rawFlowItem struct {
	ID    string     `json:"id"`
	Type  string     `json:"type"`
	Z     string     `json:"z"`
	Name  string     `json:"name"`
	Label string     `json:"label"`
	D     bool       `json:"d"`
	Wires [][]string `json:"wires"`
	Raw   json.RawMessage `json:"-"`
}

type flowComputation struct {
	source model.FlowSource
	list   model.FlowList
	detail map[string]model.FlowDetail
}

func NewFlowService(dataDir string) FlowService {
	return NewFlowServiceWithProvider(dataDir, newOllamaFlowAnalysisProviderFromEnv())
}

func NewFlowServiceWithProvider(dataDir string, provider FlowAnalysisProvider) FlowService {
	return FlowService{
		dataDir:       dataDir,
		configService: NewConfigService(dataDir, nil),
		provider:      provider,
	}
}

func (s FlowService) List() (model.FlowList, error) {
	computed, err := s.compute()
	if err != nil {
		return model.FlowList{}, err
	}
	return computed.list, nil
}

func (s FlowService) Get(id string) (model.FlowDetailResponse, error) {
	computed, err := s.compute()
	if err != nil {
		return model.FlowDetailResponse{}, err
	}

	flow, ok := computed.detail[strings.TrimSpace(id)]
	if !ok {
		return model.FlowDetailResponse{}, ErrFlowNotFound
	}

	return model.FlowDetailResponse{Source: computed.source, Flow: flow}, nil
}

// Export serializes the raw flow items for the given tab IDs (plus their
// child nodes and global config nodes) as a Node-RED-compatible flows.json byte slice.
// Returns ErrFlowNotFound if any requested ID does not exist as a tab.
func (s FlowService) Export(ids []string) ([]byte, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one flow ID required")
	}

	path, _, err := s.resolveFlowPath()
	if err != nil {
		return nil, err
	}

	items, err := s.rawFlowsFromPath(path)
	if err != nil {
		return nil, err
	}

	// Build a set of requested tab IDs and verify they exist
	requestedTabIDs := make(map[string]struct{})
	tabExists := make(map[string]bool)
	for _, item := range items {
		if item.Type == "tab" {
			tabExists[item.ID] = false
		}
	}
	for _, id := range ids {
		requestedTabIDs[id] = struct{}{}
	}

	// Check if all requested IDs exist as tabs
	for id := range requestedTabIDs {
		if _, ok := tabExists[id]; !ok {
			return nil, fmt.Errorf("flow not found: %s", id)
		}
		tabExists[id] = true
	}

	// Filter items: keep tabs in requested set, all nodes belonging to those tabs, and global config nodes
	filtered := make([]json.RawMessage, 0)
	for _, item := range items {
		// Include if it's a requested tab
		if item.Type == "tab" {
			if _, ok := requestedTabIDs[item.ID]; ok {
				filtered = append(filtered, item.Raw)
			}
			continue
		}

		// Include if it's a global config node (z == "")
		if item.Z == "" {
			filtered = append(filtered, item.Raw)
			continue
		}

		// Include if it's a child node of a requested tab
		if _, ok := requestedTabIDs[item.Z]; ok {
			filtered = append(filtered, item.Raw)
		}
	}

	result, err := json.Marshal(filtered)
	if err != nil {
		return nil, fmt.Errorf("serialize flows: %w", err)
	}
	return result, nil
}

// Import validates data as a JSON array, atomically writes it to flows.json
// via a .tmp file + os.Rename, and returns an ImportResponse.
// Caller is responsible for holding OperationLock before calling.
func (s FlowService) Import(data []byte) (model.ImportResponse, error) {
	// 1. Validate: must be a JSON array
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return model.ImportResponse{}, fmt.Errorf("invalid flows JSON: %w", err)
	}

	// 2. Resolve destination path
	path, _, err := s.resolveFlowPath()
	if err != nil {
		return model.ImportResponse{}, err
	}

	// 3. Write to .tmp (same directory = same filesystem)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return model.ImportResponse{}, fmt.Errorf("write temp flow file: %w", err)
	}

	// 4. Validate .tmp can be re-parsed (double-check disk write)
	tmpBytes, err := os.ReadFile(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return model.ImportResponse{}, fmt.Errorf("read temp flow file: %w", err)
	}

	if err := json.Unmarshal(tmpBytes, &[]json.RawMessage{}); err != nil {
		_ = os.Remove(tmpPath)
		return model.ImportResponse{}, fmt.Errorf("temp file validation failed: %w", err)
	}

	// 5. Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return model.ImportResponse{}, fmt.Errorf("atomic rename failed: %w", err)
	}

	return model.ImportResponse{
		ImportedCount:   len(items),
		Message:         "Flows imported successfully.",
		RestartAdvisory: true,
	}, nil
}

// rawFlowsFromPath reads and parses the flows file, returning items with preserved raw fields.
func (s FlowService) rawFlowsFromPath(path string) ([]rawFlowItem, error) {
	raw, err := platform.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read flow file: %w", err)
	}

	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("parse flow file: %w", err)
	}

	result := make([]rawFlowItem, len(items))
	for i, rawMsg := range items {
		var item rawFlowItem
		if err := json.Unmarshal(rawMsg, &item); err != nil {
			return nil, fmt.Errorf("parse flow item %d: %w", i, err)
		}
		item.Raw = rawMsg
		result[i] = item
	}

	return result, nil
}

func (s FlowService) Analyze(ctx context.Context, id string) (model.FlowAnalysis, error) {
	computed, err := s.compute()
	if err != nil {
		return model.FlowAnalysis{}, err
	}

	flow, ok := computed.detail[strings.TrimSpace(id)]
	if !ok {
		return model.FlowAnalysis{}, ErrFlowNotFound
	}
	if s.provider == nil {
		return model.FlowAnalysis{}, &FlowAnalysisProviderUnavailableError{
			Provider: "ollama",
			Message:  "No flow analysis provider is configured.",
			Action:   "Configure a local Ollama endpoint for advisory flow analysis.",
		}
	}

	result, err := s.provider.Analyze(ctx, buildFlowAnalysisPrompt(computed.source, flow))
	if err != nil {
		return model.FlowAnalysis{}, err
	}

	return model.FlowAnalysis{
		Source:      computed.source,
		Flow:        flow.FlowSummary,
		Advisory:    true,
		Summary:     strings.TrimSpace(result.Summary),
		Strengths:   sanitizeAnalysisItems(result.Strengths),
		Issues:      sanitizeAnalysisItems(result.Issues),
		Suggestions: sanitizeAnalysisItems(result.Suggestions),
		Provider:    s.provider.Metadata(),
	}, nil
}

func (s FlowService) compute() (flowComputation, error) {
	path, source, err := s.resolveFlowPath()
	if err != nil {
		return flowComputation{}, err
	}

	raw, err := platform.ReadFile(path)
	if err != nil {
		return flowComputation{}, fmt.Errorf("read flow file: %w", err)
	}

	var items []rawFlowItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return flowComputation{}, fmt.Errorf("parse flow file: %w", err)
	}

	tabs := make(map[string]rawFlowItem)
	tabOrder := make([]string, 0)
	for _, item := range items {
		if item.Type != "tab" || item.ID == "" {
			continue
		}
		tabs[item.ID] = item
		tabOrder = append(tabOrder, item.ID)
	}

	nodesByTab := make(map[string][]rawFlowItem)
	nodeToTab := make(map[string]string)
	for _, item := range items {
		if item.Z == "" || !isCountedFlowNode(item) {
			continue
		}
		if _, ok := tabs[item.Z]; !ok {
			continue
		}
		nodesByTab[item.Z] = append(nodesByTab[item.Z], item)
		nodeToTab[item.ID] = item.Z
	}

	inboundByTab := make(map[string]int)
	outboundByTab := make(map[string]int)
	for tabID, nodes := range nodesByTab {
		for _, node := range nodes {
			outboundByTab[tabID] += countNodeWires(node)
			for _, output := range node.Wires {
				for _, targetID := range output {
					if targetTabID, ok := nodeToTab[targetID]; ok {
						inboundByTab[targetTabID]++
					}
				}
			}
		}
	}

	totals := model.FlowSummaryTotals{}
	summaries := make([]model.FlowSummary, 0, len(tabOrder))
	details := make(map[string]model.FlowDetail, len(tabOrder))

	for _, tabID := range tabOrder {
		tab := tabs[tabID]
		nodes := append([]rawFlowItem(nil), nodesByTab[tabID]...)
		sort.Slice(nodes, func(i, j int) bool {
			left := displayName(nodes[i].Name, nodes[i].Label, nodes[i].Type, nodes[i].ID)
			right := displayName(nodes[j].Name, nodes[j].Label, nodes[j].Type, nodes[j].ID)
			if left == right {
				return nodes[i].ID < nodes[j].ID
			}
			return left < right
		})

		summary := model.FlowSummary{
			ID:                tabID,
			Label:             displayName(tab.Label, tab.Name, "Flow", tabID),
			InboundWireCount:  inboundByTab[tabID],
			OutboundWireCount: outboundByTab[tabID],
		}
		typeCounts := make(map[string]int)
		nodeSummaries := make([]model.FlowNodeSummary, 0, len(nodes))

		for _, node := range nodes {
			summary.NodeCount++
			if node.D {
				summary.DisabledNodeCount++
			}
			if isSubflowUsage(node.Type) {
				summary.SubflowUsageCount++
			} else if isCustomFlowNodeType(node.Type) {
				summary.CustomNodeCount++
			}
			typeCounts[node.Type]++
			nodeSummaries = append(nodeSummaries, model.FlowNodeSummary{
				ID:        node.ID,
				Type:      node.Type,
				Name:      displayName(node.Name, node.Label, node.Type, node.ID),
				Disabled:  node.D,
				WireCount: countNodeWires(node),
			})
		}

		typeMetrics := make([]model.FlowTypeMetric, 0, len(typeCounts))
		for nodeType, count := range typeCounts {
			typeMetrics = append(typeMetrics, model.FlowTypeMetric{
				Type:   nodeType,
				Count:  count,
				Custom: !isSubflowUsage(nodeType) && isCustomFlowNodeType(nodeType),
			})
		}
		sort.Slice(typeMetrics, func(i, j int) bool {
			if typeMetrics[i].Count == typeMetrics[j].Count {
				return typeMetrics[i].Type < typeMetrics[j].Type
			}
			return typeMetrics[i].Count > typeMetrics[j].Count
		})

		summaries = append(summaries, summary)
		details[tabID] = model.FlowDetail{FlowSummary: summary, NodeTypes: typeMetrics, Nodes: nodeSummaries}

		totals.FlowCount++
		totals.NodeCount += summary.NodeCount
		totals.DisabledNodeCount += summary.DisabledNodeCount
		totals.CustomNodeCount += summary.CustomNodeCount
		totals.InboundWireCount += summary.InboundWireCount
		totals.OutboundWireCount += summary.OutboundWireCount
		totals.SubflowUsageCount += summary.SubflowUsageCount
	}

	return flowComputation{
		source: source,
		list: model.FlowList{
			Source:  source,
			Summary: totals,
			Items:   summaries,
		},
		detail: details,
	}, nil
}

func (s FlowService) resolveFlowPath() (string, model.FlowSource, error) {
	cfg, err := s.configService.LoadFullConfig()
	if err != nil {
		return "", model.FlowSource{}, fmt.Errorf("load config: %w", err)
	}

	userDir := strings.TrimSpace(cfg.Flows.UserDir)
	if userDir == "" {
		userDir = filepath.Join(s.dataDir, "nodered")
	}
	flowFile := strings.TrimSpace(cfg.Flows.FlowFile)
	if flowFile == "" {
		flowFile = model.DefaultFullAppConfig().Flows.FlowFile
	}
	path := filepath.Join(userDir, flowFile)

	source := model.FlowSource{UserDir: userDir, FlowFile: flowFile, Path: path, ReadOnly: true}
	if stat, err := os.Stat(path); err == nil {
		source.UpdatedAt = stat.ModTime().UTC().Format(time.RFC3339)
	}

	return path, source, nil
}

func isCountedFlowNode(item rawFlowItem) bool {
	if item.ID == "" || item.Type == "" {
		return false
	}
	return item.Type != "group" && item.Type != "tab" && item.Type != "subflow"
}

func isSubflowUsage(nodeType string) bool {
	return strings.HasPrefix(nodeType, "subflow:")
}

func isCustomFlowNodeType(nodeType string) bool {
	if _, ok := builtInFlowNodeTypes[nodeType]; ok {
		return false
	}
	return nodeType != "" && !isSubflowUsage(nodeType)
}

func countNodeWires(node rawFlowItem) int {
	total := 0
	for _, output := range node.Wires {
		total += len(output)
	}
	return total
}

func displayName(primary, secondary, fallback, id string) string {
	if value := strings.TrimSpace(primary); value != "" {
		return value
	}
	if value := strings.TrimSpace(secondary); value != "" {
		return value
	}
	if value := strings.TrimSpace(fallback); value != "" {
		return value
	}
	return id
}

func buildFlowAnalysisPrompt(source model.FlowSource, flow model.FlowDetail) string {
	nodeTypes := make([]string, 0, min(6, len(flow.NodeTypes)))
	for i, item := range flow.NodeTypes {
		if i >= 6 {
			break
		}
		label := fmt.Sprintf("%s:%d", item.Type, item.Count)
		if item.Custom {
			label += " (custom)"
		}
		nodeTypes = append(nodeTypes, label)
	}

	nodes := make([]string, 0, min(8, len(flow.Nodes)))
	for i, node := range flow.Nodes {
		if i >= 8 {
			break
		}
		details := []string{node.Type}
		if node.Disabled {
			details = append(details, "disabled")
		}
		if node.WireCount > 0 {
			details = append(details, fmt.Sprintf("wires:%d", node.WireCount))
		}
		nodes = append(nodes, fmt.Sprintf("%s [%s]", node.Name, strings.Join(details, ", ")))
	}

	return strings.TrimSpace(fmt.Sprintf(`You are analyzing a single Node-RED flow for an operator UI.
Respond with STRICT JSON only using this shape:
{"summary":"string","strengths":["..."],"issues":["..."],"suggestions":["..."]}

Rules:
- Write in Spanish.
- Keep the answer advisory and operator-friendly.
- Do not invent hidden runtime behavior.
- Do not suggest auto-fixes, code generation, or multi-flow actions.
- summary: 1-3 short sentences.
- strengths/issues/suggestions: 1-4 concise strings each.

Flow source:
- path: %s
- updated_at: %s

Flow metrics:
- id: %s
- label: %s
- node_count: %d
- disabled_node_count: %d
- custom_node_count: %d
- inbound_wire_count: %d
- outbound_wire_count: %d
- subflow_usage_count: %d
- top_node_types: %s
- sample_nodes: %s
`,
		source.Path,
		source.UpdatedAt,
		flow.ID,
		flow.Label,
		flow.NodeCount,
		flow.DisabledNodeCount,
		flow.CustomNodeCount,
		flow.InboundWireCount,
		flow.OutboundWireCount,
		flow.SubflowUsageCount,
		joinOrFallback(nodeTypes, "none"),
		joinOrFallback(nodes, "none"),
	))
}

func sanitizeAnalysisItems(items []string) []string {
	clean := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		item = strings.TrimPrefix(item, "- ")
		item = strings.TrimPrefix(item, "* ")
		if item != "" {
			clean = append(clean, item)
		}
	}
	return clean
}

func joinOrFallback(items []string, fallback string) string {
	if len(items) == 0 {
		return fallback
	}
	return strings.Join(items, "; ")
}

type ollamaFlowAnalysisProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

func newOllamaFlowAnalysisProviderFromEnv() *ollamaFlowAnalysisProvider {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("NRCC_OLLAMA_URL")), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:11434"
	}
	model := strings.TrimSpace(os.Getenv("NRCC_FLOW_ANALYSIS_MODEL"))
	if model == "" {
		model = "llama3.2"
	}

	return &ollamaFlowAnalysisProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 45 * time.Second},
	}
}

func (p *ollamaFlowAnalysisProvider) Metadata() model.FlowAnalysisProvider {
	return model.FlowAnalysisProvider{Name: "ollama", Model: p.model, Local: true}
}

func (p *ollamaFlowAnalysisProvider) Analyze(ctx context.Context, prompt string) (FlowAnalysisPayload, error) {
	body, err := json.Marshal(map[string]any{
		"model":  p.model,
		"prompt": prompt,
		"stream": false,
		"format": "json",
	})
	if err != nil {
		return FlowAnalysisPayload{}, fmt.Errorf("encode ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return FlowAnalysisPayload{}, fmt.Errorf("build ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		if isFlowAnalysisConnectionError(err) {
			return FlowAnalysisPayload{}, &FlowAnalysisProviderUnavailableError{
				Provider: "ollama",
				Model:    p.model,
				Message:  "Ollama is not reachable for flow analysis.",
				Action:   fmt.Sprintf("Start Ollama locally and pull the model with 'ollama pull %s'.", p.model),
				Cause:    err,
			}
		}
		return FlowAnalysisPayload{}, fmt.Errorf("request flow analysis: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return FlowAnalysisPayload{}, fmt.Errorf("read ollama response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return FlowAnalysisPayload{}, &FlowAnalysisProviderUnavailableError{
			Provider: "ollama",
			Model:    p.model,
			Message:  "Ollama rejected the flow analysis request.",
			Action:   fmt.Sprintf("Confirm the model '%s' is installed and the Ollama API is healthy.", p.model),
			Cause:    fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw))),
		}
	}

	var envelope struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return FlowAnalysisPayload{}, fmt.Errorf("decode ollama response: %w", err)
	}

	var result FlowAnalysisPayload
	if err := json.Unmarshal([]byte(envelope.Response), &result); err != nil {
		return FlowAnalysisPayload{}, fmt.Errorf("decode flow analysis payload: %w", err)
	}

	return result, nil
}

func isFlowAnalysisConnectionError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr)
}
