package model

// RuntimeStatus represents the current status of the Node-RED runtime
type RuntimeStatus struct {
	Status              string           `json:"status"` // running, stopped, error, detected
	PID                 int              `json:"pid,omitempty"`
	Uptime              int64            `json:"uptime"`              // seconds
	RestartCount        int              `json:"restartCount"`        // durable cumulative auto-restart count
	ConsecutiveFailures int              `json:"consecutiveFailures"` // backoff/give-up counter since last user start
	Memory              *ProcessMemory   `json:"memory,omitempty"`
	Version             string           `json:"version,omitempty"`
	StartedAt           string           `json:"startedAt,omitempty"`
	InstallationMode    InstallationMode `json:"installationMode,omitempty"`
	ManagedByNRCC       bool             `json:"managedByNrcc,omitempty"`
	Detected            bool             `json:"detected,omitempty"`
}

// ProcessMemory represents memory usage of the Node-RED process
type ProcessMemory struct {
	RSS       int64 `json:"rss"` // resident set size
	HeapTotal int64 `json:"heapTotal"`
	HeapUsed  int64 `json:"heapUsed"`
	External  int64 `json:"external"`
}

// LogEntry represents a single log line
type LogEntry struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"` // ISO-8601
	Level     string `json:"level"`     // info, warn, error, debug
	Source    string `json:"source"`    // stdout, stderr
	Message   string `json:"message"`
}
