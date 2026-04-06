package model

type ManagedEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ManagedEnvState struct {
	Variables       []ManagedEnvVar `json:"variables"`
	RestartRequired bool            `json:"restartRequired"`
}
