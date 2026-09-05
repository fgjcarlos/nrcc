package model

import (
	"encoding/json"
)

// NodeRedConfig represents the Node-RED configuration
type NodeRedConfig struct {
	// Basic settings
	Port             int    `json:"port,omitempty"`
	UIPort           int    `json:"uiPort,omitempty"` // Frontend uses this
	UIHost           string `json:"uiHost,omitempty"`
	HTTPAdminRoot    string `json:"httpAdminRoot,omitempty"`
	HTTPNodeRoot     string `json:"httpNodeRoot,omitempty"`
	FlowFile         string `json:"flowFile,omitempty"`
	UserDir          string `json:"userDir,omitempty"`
	NodesDir         string `json:"nodesDir,omitempty"`
	CredentialSecret string `json:"credentialSecret,omitempty"`
	DisableEditor    bool   `json:"disableEditor,omitempty"`

	// Authentication
	AdminAuth    *AdminAuth  `json:"adminAuth,omitempty"`
	NodeHttpAuth interface{} `json:"nodeHttpAuth,omitempty"`
	StaticAuth   interface{} `json:"staticAuth,omitempty"`

	// Projects & Logging
	ProjectsEnabled bool        `json:"projectsEnabled,omitempty"`
	Logging         interface{} `json:"logging,omitempty"`

	// TLS (issue #762 slice 1). HTTPS enables a TLS listener; RequireHttps
	// redirects plain-http traffic to https when set. Both require a Node-RED
	// restart. Https carries the on-disk paths to the PEM key, cert and
	// optional CA; the backend renders them as fs.readFileSync() so the
	// operator never has to inline certificate content.
	RequireHttps bool      `json:"requireHttps,omitempty"`
	Https        *HttpsConfig `json:"https,omitempty"`

	// Editor Theme
	EditorTheme interface{} `json:"editorTheme,omitempty"`

	// Runtime State
	RuntimeState interface{} `json:"runtimeState,omitempty"`

	// Language
	Lang string `json:"lang,omitempty"`

	// Legacy
	FunctionLibraries []FunctionLibrary      `json:"functionLibraries,omitempty"`
	EnvVars           []EnvVar               `json:"envVars,omitempty"`
	SettingsPath      string                 `json:"settingsPath,omitempty"`
	SettingsSource    string                 `json:"settingsSource,omitempty"`
	Extra             map[string]interface{} `json:"-"`
}

// AdminAuth represents Node-RED admin authentication config
type AdminAuth struct {
	Type  string          `json:"type"`
	Users []AdminAuthUser `json:"users,omitempty"`
}

// AdminAuthUser represents a user in adminAuth config
type AdminAuthUser struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Permissions string `json:"permissions"`
}

// HttpsConfig describes the Node-RED `https` block. Node-RED reads each
// entry through fs.readFileSync at startup, so storing on-disk paths is the
// safest representation: we never embed certificate bytes in nrcc's own JSON
// store, and operator rotation is "replace the file, restart Node-RED".
type HttpsConfig struct {
	Key        string `json:"key,omitempty"`
	Cert       string `json:"cert,omitempty"`
	CA         string `json:"ca,omitempty"`
	Port       int    `json:"port,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

// FunctionLibrary represents a Node-RED function library
type FunctionLibrary struct {
	Name    string `json:"name"`
	Alias   string `json:"alias,omitempty"`
	Version string `json:"version,omitempty"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type"` // Type vocabulary: "string" | "number" | "boolean" | "secret"
	Description string `json:"description,omitempty"`
	Encrypted   bool   `json:"encrypted,omitempty"`
	Source      string `json:"source,omitempty"` // "nrcc" takes precedence over "node-red" during imports.
}

// UnmarshalJSON handles both uiPort and port field names
func (c *NodeRedConfig) UnmarshalJSON(data []byte) error {
	type Alias NodeRedConfig
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// If uiPort was provided but port wasn't, use uiPort
	if c.UIPort > 0 && c.Port == 0 {
		c.Port = c.UIPort
	}

	return nil
}

// MarshalJSON ensures we export port consistently
func (c NodeRedConfig) MarshalJSON() ([]byte, error) {
	type Alias NodeRedConfig
	return json.Marshal(&struct {
		*Alias
		Port int `json:"port"`
	}{
		Alias: (*Alias)(&c),
		Port:  c.Port,
	})
}

// DefaultNodeRedConfig returns default configuration values
func DefaultNodeRedConfig() NodeRedConfig {
	return NodeRedConfig{
		FlowFile:      "flows.json",
		UserDir:       "./data",
		Port:          1880,
		UIPort:        1880,
		HTTPAdminRoot: "/",
		HTTPNodeRoot:  "/",
	}
}
