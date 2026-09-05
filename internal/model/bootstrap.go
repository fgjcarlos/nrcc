package model

// InstallationMode describes how Node-RED is installed or expected to run.
type InstallationMode string

const (
	InstallationModeNone    InstallationMode = "none"
	InstallationModeNative  InstallationMode = "native"
	InstallationModeDocker  InstallationMode = "docker"
	InstallationModeUnknown InstallationMode = "unknown"
)

// DependencyStatus describes the presence of a host dependency.
type DependencyStatus struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	Command   string `json:"command,omitempty"`
	Details   string `json:"details,omitempty"`
}

// NodeRedEnvironment describes the detected Node-RED installation.
type NodeRedEnvironment struct {
	Detected      bool             `json:"detected"`
	Mode          InstallationMode `json:"mode"`
	ManagedByNRCC bool             `json:"managedByNrcc"`
	Running       bool             `json:"running"`
	Version       string           `json:"version,omitempty"`
	Executable    string           `json:"executable,omitempty"`
	ContainerName string           `json:"containerName,omitempty"`
	ContainerID   string           `json:"containerId,omitempty"`
	UserDir       string           `json:"userDir,omitempty"`
	SettingsPath  string           `json:"settingsPath,omitempty"`
}

// SettingsDocument describes the active settings.js file and backup metadata.
type SettingsDocument struct {
	Path       string `json:"path"`
	Source     string `json:"source"`
	Writable   bool   `json:"writable"`
	BackupPath string `json:"backupPath,omitempty"`
	Content    string `json:"content,omitempty"`
}

// ConfigurationCapabilities describes whether NRCC can safely edit the
// detected Node-RED settings.js contract.
type ConfigurationCapabilities struct {
	RuntimeVersion string `json:"runtimeVersion"`
	Adapter        string `json:"adapter"`
	CatalogVersion string `json:"catalogVersion"`
	Source         string `json:"source"`
	Mode           string `json:"mode"`
	Editable       bool   `json:"editable"`
	Reason         string `json:"reason,omitempty"`
}

// HostStatus summarizes the detected host and runtime environment.
type HostStatus struct {
	Platform        string                   `json:"platform"`
	Ready           bool                     `json:"ready"`
	Interactive     bool                     `json:"interactive"`
	NodeJS          DependencyStatus         `json:"nodejs"`
	NPM             DependencyStatus         `json:"npm"`
	NodeRedBinary   DependencyStatus         `json:"nodeRedBinary"`
	Portless        DependencyStatus         `json:"portless"`
	Docker          DependencyStatus         `json:"docker"`
	DockerCompose   DependencyStatus         `json:"dockerCompose"`
	NodeRed         NodeRedEnvironment       `json:"nodeRed"`
	Settings        SettingsDocument         `json:"settings"`
	Configuration   ConfigurationCapabilities `json:"configuration"`
	Recommendations []string                 `json:"recommendations,omitempty"`
}
