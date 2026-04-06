package model

type AppConfig struct {
	HTTPAdminRoot      string `json:"httpAdminRoot"`
	FlowFile           string `json:"flowFile"`
	DiagnosticsEnabled bool   `json:"diagnosticsEnabled"`
	ProjectsEnabled    bool   `json:"projectsEnabled"`
	CredentialSecret   string `json:"credentialSecret"`
}

type ConfigValidationResult struct {
	Valid           bool              `json:"valid"`
	RestartRequired bool              `json:"restartRequired"`
	Errors          []string          `json:"errors"`
	Diff            []ConfigDiffEntry `json:"diff"`
}

type ConfigDiffEntry struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}
