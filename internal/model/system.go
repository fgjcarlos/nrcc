package model

type LocalAccessStatus struct {
	Mode              string `json:"mode"`
	Hostname          string `json:"hostname,omitempty"`
	URL               string `json:"url"`
	FallbackURL       string `json:"fallbackUrl"`
	PortlessAvailable bool   `json:"portlessAvailable"`
	Configured        bool   `json:"configured"`
	Operational       bool   `json:"operational"`
	Message           string `json:"message"`
}

type SystemInfo struct {
	GOOS        string            `json:"goos"`
	GOARCH      string            `json:"goarch"`
	CPUs        int               `json:"cpus"`
	Hostname    string            `json:"hostname"`
	Timestamp   string            `json:"timestamp"`
	LocalAccess LocalAccessStatus `json:"localAccess"`
}
