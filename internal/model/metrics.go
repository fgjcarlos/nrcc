package model

// MetricsSnapshot represents a point-in-time snapshot of system metrics.
type MetricsSnapshot struct {
	Timestamp     string  `json:"timestamp"`
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryPercent float64 `json:"memoryPercent"`
	DiskPercent   float64 `json:"diskPercent"`
}

// RestartEvent records a single Node-RED restart with context information.
type RestartEvent struct {
	Timestamp   string `json:"timestamp"`
	ExitCode    int    `json:"exitCode"`
	Attempt     int    `json:"attempt"`
	MaxAttempts int    `json:"maxAttempts"`
}
