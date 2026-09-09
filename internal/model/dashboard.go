package model

// DiscoverySource reports whether an input file could be used for discovery.
type DiscoverySource struct {
	State string `json:"state"`
}

// DashboardPackage identifies an installed dashboard package.
type DashboardPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// LegacyDashboard identifies the legacy package and its default dashboard path.
type LegacyDashboard struct {
	Package DashboardPackage `json:"package"`
	Path    string           `json:"path"`
}

// DashboardPath identifies a dashboard path from its owning flow node.
type DashboardPath struct {
	NodeID string `json:"nodeId"`
	Path   string `json:"path,omitempty"`
	State  string `json:"state"`
}

// DashboardDiscovery is the read-only dashboard capability contract.
type DashboardDiscovery struct {
	Packages DiscoverySource    `json:"packages"`
	Flows    DiscoverySource    `json:"flows"`
	Legacy   *LegacyDashboard   `json:"legacy,omitempty"`
	FlowFuse []DashboardPackage `json:"flowFuse"`
	UIBases  []DashboardPath    `json:"uiBases"`
}

// DashboardAccessPolicy selects a reviewed dashboard access recipe.
// Secrets are write-only and are never returned in API responses.
type DashboardAccessPolicy struct {
	Target           string `json:"target"`
	Recipe           string `json:"recipe"`
	Username         string `json:"username,omitempty"`
	Secret           string `json:"secret,omitempty"`
	ExpectedRevision string `json:"expectedRevision,omitempty"`
}
