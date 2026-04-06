package model

type RuntimeStatus struct {
	Running    bool   `json:"running"`
	Healthy    bool   `json:"healthy"`
	PID        int    `json:"pid"`
	Port       int    `json:"port"`
	UptimeSec  int64  `json:"uptimeSec"`
	Version    string `json:"version,omitempty"`
	DataDir    string `json:"dataDir"`
	LastError  string `json:"lastError,omitempty"`
	LastExit   string `json:"lastExit,omitempty"`
	StartedAt  string `json:"startedAt,omitempty"`
	BinaryPath string `json:"binaryPath,omitempty"`
}
