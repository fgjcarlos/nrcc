package service

import (
	"regexp"

	"github.com/fgjcarlos/nrcc/internal/model"
)

const nodeRED5CatalogVersion = "5.0.6"

var nodeREDMajorVersionPattern = regexp.MustCompile(`^v?([0-9]+)(?:\.[0-9]+){0,2}$`)

// SettingCatalogEntry identifies a canonical Node-RED setting that NRCC can
// present through its structured configuration experience.
type SettingCatalogEntry struct {
	Key              string
	Shape            string
	Default          string
	Validation       string
	Secret           bool
	RestartRequired  bool
	UIEditable       bool
}

var nodeRED5Catalog = []SettingCatalogEntry{
	{Key: "flowFile", Shape: "string", Default: "flows.json", Validation: "non-empty", RestartRequired: true, UIEditable: true},
	{Key: "credentialSecret", Shape: "string-or-false", Default: "generated", Validation: "string-or-false", Secret: true, RestartRequired: true, UIEditable: true},
	{Key: "flowFilePretty", Shape: "boolean", Default: "true", Validation: "boolean", RestartRequired: true, UIEditable: true},
	{Key: "userDir", Shape: "string", Default: "~/.node-red", Validation: "non-empty-path", RestartRequired: true, UIEditable: true},
	{Key: "nodesDir", Shape: "string", Validation: "path", RestartRequired: true, UIEditable: true},
	{Key: "adminAuth", Shape: "object", Validation: "credentials-or-strategy", Secret: true, RestartRequired: true, UIEditable: true},
	{Key: "httpNodeAuth", Shape: "object", Validation: "bcrypt-password", Secret: true, RestartRequired: true, UIEditable: true},
	{Key: "httpStaticAuth", Shape: "object", Validation: "bcrypt-password", Secret: true, RestartRequired: true, UIEditable: true},
	{Key: "uiPort", Shape: "number-or-expression", Default: "process.env.PORT || 1880", Validation: "port-or-expression", RestartRequired: true, UIEditable: true},
	{Key: "uiHost", Shape: "string", Default: "0.0.0.0", Validation: "host-or-ip", RestartRequired: true, UIEditable: true},
	{Key: "httpAdminRoot", Shape: "string-or-false", Default: "/", Validation: "path-or-false", RestartRequired: true, UIEditable: true},
	{Key: "httpNodeRoot", Shape: "string-or-false", Default: "/", Validation: "path-or-false", RestartRequired: true, UIEditable: true},
	{Key: "https", Shape: "https-options", Default: "undefined", Validation: "https-options", Secret: true, RestartRequired: true, UIEditable: true},
	{Key: "requireHttps", Shape: "boolean", Default: "false", Validation: "boolean", RestartRequired: true, UIEditable: true},
	{Key: "httpStatic", Shape: "string-or-array", Validation: "path-or-static-sources", RestartRequired: true, UIEditable: false},
	{Key: "lang", Shape: "string", Default: "en-US", Validation: "locale", RestartRequired: true, UIEditable: true},
	{Key: "runtimeState", Shape: "object", Default: "{enabled:false,ui:false}", Validation: "runtime-state-options", RestartRequired: true, UIEditable: true},
	{Key: "logging", Shape: "object", Default: "{console:{level:info}}", Validation: "logging-options", RestartRequired: true, UIEditable: true},
	{Key: "disableEditor", Shape: "boolean", Default: "false", Validation: "boolean", RestartRequired: true, UIEditable: true},
	{Key: "editorTheme", Shape: "object", Validation: "editor-theme-options", RestartRequired: true, UIEditable: true},
	{Key: "functionGlobalContext", Shape: "object", Default: "{}", Validation: "object", Secret: true, RestartRequired: true, UIEditable: false},
}

// NodeRED5Catalog returns a copy of the Node-RED 5.0.6 canonical setting
// catalog. The catalog is deliberately separate from the UI so future adapters
// must opt in to every supported setting shape.
//
// Per-entry documentation (issue #762 acceptance: every control states its
// Node-RED 5 meaning, default, restart impact, secret behavior and
// verification method):
//
//   - credentialSecret: encrypts credentials.json at rest. "false" disables
//     encryption (not recommended). Restart required. Secret because changing
//     it invalidates stored credentials. Verification: editor login still
//     works after restart and credentials.json cannot be read without the
//     secret.
//   - https: TLS listener block (key/cert/ca/port/passphrase options). When
//     set Node-RED serves HTTPS instead of HTTP. Restart required. Secret
//     because cert/key paths leak server identity. Verification: openssl
//     s_client -connect host:port returns the configured certificate.
//   - requireHttps: when true Node-RED redirects http:// requests to https://.
//     Restart required. Verification: editor URL without https redirects to
//     https://<same path>.
func NodeRED5Catalog() []SettingCatalogEntry {
	return append([]SettingCatalogEntry(nil), nodeRED5Catalog...)
}

// ResolveConfigurationCapabilities applies the Node-RED compatibility policy:
// v4 is migration/read-only, 5.x is editable when settings.js is writable, and
// future or unknown versions remain read-only until an adapter is shipped.
func ResolveConfigurationCapabilities(version string, settings model.SettingsDocument) model.ConfigurationCapabilities {
	capabilities := model.ConfigurationCapabilities{
		RuntimeVersion: version,
		CatalogVersion: nodeRED5CatalogVersion,
		Source:         settings.Source,
		Mode:           "read-only",
		Adapter:        "unsupported",
	}
	if capabilities.RuntimeVersion == "" {
		capabilities.RuntimeVersion = "unknown"
	}

	match := nodeREDMajorVersionPattern.FindStringSubmatch(version)
	if len(match) != 2 {
		capabilities.Reason = "Node-RED version is unknown; configuration editing is disabled until a compatible adapter is available."
		return capabilities
	}

	switch match[1] {
	case "4":
		capabilities.Adapter = "nodered-4-read-only"
		capabilities.Reason = "Node-RED 4 is available in migration read-only mode."
	case "5":
		capabilities.Adapter = "nodered-5"
		if !settings.Writable {
			capabilities.Reason = "The Node-RED 5 settings source is not writable."
			return capabilities
		}
		capabilities.Mode = "editable"
		capabilities.Editable = true
	case "6":
		capabilities.Reason = "Node-RED 6 and later are read-only until a compatible adapter is available."
	default:
		capabilities.Reason = "This Node-RED version is not supported for configuration editing."
	}
	return capabilities
}
