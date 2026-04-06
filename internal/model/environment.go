package model

type CheckStatus string

const (
	StatusOK      CheckStatus = "ok"
	StatusWarn    CheckStatus = "warn"
	StatusFail    CheckStatus = "fail"
	StatusSkipped CheckStatus = "skipped"
)

type EnvironmentCheck struct {
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Detail  string      `json:"detail"`
	Command string      `json:"command,omitempty"`
}

type EnvironmentReport struct {
	OS              string             `json:"os"`
	Arch            string             `json:"arch"`
	DataDir         string             `json:"dataDir"`
	NodeInstalled   bool               `json:"nodeInstalled"`
	NPMInstalled    bool               `json:"npmInstalled"`
	PortlessPresent bool               `json:"portlessPresent"`
	NodeRedReady    bool               `json:"nodeRedReady"`
	Checks          []EnvironmentCheck `json:"checks"`
}
