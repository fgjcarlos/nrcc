package model

type FlowSource struct {
	UserDir   string `json:"userDir"`
	FlowFile  string `json:"flowFile"`
	Path      string `json:"path"`
	ReadOnly  bool   `json:"readOnly"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type FlowTypeMetric struct {
	Type   string `json:"type"`
	Count  int    `json:"count"`
	Custom bool   `json:"custom"`
}

type FlowNodeSummary struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Disabled  bool   `json:"disabled"`
	WireCount int    `json:"wireCount"`
}

type FlowSummary struct {
	ID                string `json:"id"`
	Label             string `json:"label"`
	NodeCount         int    `json:"nodeCount"`
	DisabledNodeCount int    `json:"disabledNodeCount"`
	CustomNodeCount   int    `json:"customNodeCount"`
	InboundWireCount  int    `json:"inboundWireCount"`
	OutboundWireCount int    `json:"outboundWireCount"`
	SubflowUsageCount int    `json:"subflowUsageCount"`
}

type FlowSummaryTotals struct {
	FlowCount         int `json:"flowCount"`
	NodeCount         int `json:"nodeCount"`
	DisabledNodeCount int `json:"disabledNodeCount"`
	CustomNodeCount   int `json:"customNodeCount"`
	InboundWireCount  int `json:"inboundWireCount"`
	OutboundWireCount int `json:"outboundWireCount"`
	SubflowUsageCount int `json:"subflowUsageCount"`
}

type FlowList struct {
	Source  FlowSource        `json:"source"`
	Summary FlowSummaryTotals `json:"summary"`
	Items   []FlowSummary     `json:"items"`
}

type FlowDetail struct {
	FlowSummary
	NodeTypes []FlowTypeMetric  `json:"nodeTypes"`
	Nodes     []FlowNodeSummary `json:"nodes"`
}

type FlowDetailResponse struct {
	Source FlowSource `json:"source"`
	Flow   FlowDetail `json:"flow"`
}

type FlowAnalysisProvider struct {
	Name  string `json:"name"`
	Model string `json:"model"`
	Local bool   `json:"local"`
}

type FlowAnalysis struct {
	Source      FlowSource           `json:"source"`
	Flow        FlowSummary          `json:"flow"`
	Advisory    bool                 `json:"advisory"`
	Summary     string               `json:"summary"`
	Strengths   []string             `json:"strengths"`
	Issues      []string             `json:"issues"`
	Suggestions []string             `json:"suggestions"`
	Provider    FlowAnalysisProvider `json:"provider"`
}
