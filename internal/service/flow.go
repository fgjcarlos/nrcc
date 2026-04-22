package service

import (
	"encoding/json"
	"errors"
	"fmt"
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
}

type rawFlowItem struct {
	ID    string     `json:"id"`
	Type  string     `json:"type"`
	Z     string     `json:"z"`
	Name  string     `json:"name"`
	Label string     `json:"label"`
	D     bool       `json:"d"`
	Wires [][]string `json:"wires"`
}

type flowComputation struct {
	source model.FlowSource
	list   model.FlowList
	detail map[string]model.FlowDetail
}

func NewFlowService(dataDir string) FlowService {
	return FlowService{
		dataDir:       dataDir,
		configService: NewConfigService(dataDir, nil),
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
