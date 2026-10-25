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
	AdminAuth      *AdminAuth     `json:"adminAuth,omitempty"`
	HTTPNodeAuth   *HTTPBasicAuth `json:"httpNodeAuth,omitempty"`
	HTTPStaticAuth *HTTPBasicAuth `json:"httpStaticAuth,omitempty"`

	// Projects & Logging
	ProjectsEnabled bool        `json:"projectsEnabled,omitempty"`
	Logging         interface{} `json:"logging,omitempty"`

	// TLS (issue #762 slice 1). HTTPS enables a TLS listener; RequireHttps
	// redirects plain-http traffic to https when set. Both require a Node-RED
	// restart. Https carries the on-disk paths to the PEM key, cert and
	// optional CA; the backend renders them as fs.readFileSync() so the
	// operator never has to inline certificate content.
	RequireHttps bool         `json:"requireHttps,omitempty"`
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

	// presence records which security surfaces the JSON payload that
	// produced this config carried. It is never serialized.
	presence securitySurfacePresence
}

// securitySurfacePresence tracks whether a decoded payload carried the
// authentication surfaces.
//
// Issue #881: a payload that omits `adminAuth` means "leave the surface
// alone", not "clear the surface". The zero value of a pointer field
// cannot express that difference, so the presence is recorded while
// decoding.
//
// A config built in Go (not decoded from JSON) stays authoritative: the
// caller passed the whole model on purpose, so every surface it carries
// wins.
type securitySurfacePresence struct {
	decodedFromJSON      bool
	adminAuthAbsent      bool
	httpNodeAuthAbsent   bool
	httpStaticAuthAbsent bool
}

// AdminAuthCarried reports whether the caller supplied an adminAuth
// surface. False means the payload omitted it and the loaded surface must
// be preserved.
func (c NodeRedConfig) AdminAuthCarried() bool {
	return !c.presence.decodedFromJSON || !c.presence.adminAuthAbsent
}

// HTTPNodeAuthCarried reports whether the caller supplied an httpNodeAuth
// surface. False means the payload omitted it and the loaded surface must
// be preserved.
func (c NodeRedConfig) HTTPNodeAuthCarried() bool {
	return !c.presence.decodedFromJSON || !c.presence.httpNodeAuthAbsent
}

// HTTPStaticAuthCarried reports whether the caller supplied an
// httpStaticAuth surface. False means the payload omitted it and the
// loaded surface must be preserved.
func (c NodeRedConfig) HTTPStaticAuthCarried() bool {
	return !c.presence.decodedFromJSON || !c.presence.httpStaticAuthAbsent
}

// AdminAuth represents Node-RED admin authentication config
type AdminAuth struct {
	Type              string          `json:"type"`
	Users             []AdminAuthUser `json:"users,omitempty"`
	SessionExpiryTime int             `json:"sessionExpiryTime,omitempty"`
}

// AdminAuthUser represents a user in adminAuth config
type AdminAuthUser struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Permissions string `json:"permissions"`
}

// HTTPBasicAuth is the canonical Node-RED basic-auth shape used by both
// httpNodeAuth and httpStaticAuth. Node-RED expects `user` and `pass`; `pass`
// must be a bcrypt hash. Legacy aliases are intentionally unsupported.
type HTTPBasicAuth struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

// AuthenticationSurfaces is the parsed set of independently configured
// Node-RED authentication boundaries. It is separate from NRCC accounts.
type AuthenticationSurfaces struct {
	AdminAuth      *AdminAuth
	HTTPNodeAuth   *HTTPBasicAuth
	HTTPStaticAuth *HTTPBasicAuth
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

	// Record which security surfaces this payload carried (issue #881).
	// "absent" and "explicitly null" are different intents: absent means
	// "leave the loaded surface alone", null means "clear the surface".
	c.presence = securitySurfacePresence{decodedFromJSON: true}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		c.presence.adminAuthAbsent = !hasJSONKey(raw, "adminAuth")
		c.presence.httpNodeAuthAbsent = !hasJSONKey(raw, "httpNodeAuth")
		c.presence.httpStaticAuthAbsent = !hasJSONKey(raw, "httpStaticAuth")
	}

	return nil
}

// hasJSONKey reports whether the decoded object contains the key.
func hasJSONKey(raw map[string]json.RawMessage, key string) bool {
	_, ok := raw[key]
	return ok
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
