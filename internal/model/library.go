package model

// LibraryInfo represents an npm package library
type LibraryInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Author      string   `json:"author,omitempty"`
	License     string   `json:"license,omitempty"`
}

// UpdateStatus represents the update status for Node-RED
// Deprecated: Use model.UpdateCacheEntry instead (internal/model/update.go).
// This type is retained for backward compatibility with internal APIs.
type UpdateStatus struct {
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
}

// DockerStatus represents Docker information
type DockerStatus struct {
	Available bool   `json:"available"`
	Message   string `json:"message"`
}

// FlowNode represents a Node-RED flow node
type FlowNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Label    string                 `json:"label,omitempty"`
	Position []int                  `json:"pos,omitempty"`
	Size     []int                  `json:"size,omitempty"`
	Props    map[string]interface{} `json:"props,omitempty"`
}

// Flow represents a Node-RED flow
type Flow struct {
	ID    string     `json:"id"`
	Label string     `json:"label,omitempty"`
	Nodes []FlowNode `json:"nodes,omitempty"`
}
