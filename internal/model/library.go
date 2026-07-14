package model

// LibraryInfo represents an npm package library
type LibraryInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Category    string   `json:"category,omitempty"`
	Author      string   `json:"author,omitempty"`
	License     string   `json:"license,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	Repository  string   `json:"repository,omitempty"`
	NPM         string   `json:"npm,omitempty"`
	Downloads   int      `json:"downloads,omitempty"`
	Date        string   `json:"date,omitempty"`
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
