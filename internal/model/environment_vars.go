package model

type ManagedEnvVar struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Secret   bool   `json:"secret,omitempty"`
	HasValue bool   `json:"hasValue,omitempty"`
}

type ManagedEnvState struct {
	Variables       []ManagedEnvVar `json:"variables"`
	RestartRequired bool            `json:"restartRequired"`
}
