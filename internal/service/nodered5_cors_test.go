package service

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TestParseHTTPNodeCorsBlockFromJS covers the modelable shapes of the
// Node-RED `httpNodeCors` block (spec S2/S3): string and array literals for
// origin/methods/allowedHeaders, plus the credentials boolean.
func TestParseHTTPNodeCorsBlockFromJS(t *testing.T) {
	cases := []struct {
		name string
		body string
		want *model.HTTPNodeCorsConfig
	}{
		{
			name: "absent key",
			body: "module.exports = {\n  uiPort: 1880,\n}\n",
			want: nil,
		},
		{
			name: "wildcard origin string",
			body: `  httpNodeCors: { origin: "*" },`,
			want: &model.HTTPNodeCorsConfig{Origin: []string{"*"}},
		},
		{
			name: "single quoted origin",
			body: `  httpNodeCors: { origin: 'https://app.example.com' },`,
			want: &model.HTTPNodeCorsConfig{Origin: []string{"https://app.example.com"}},
		},
		{
			name: "origin array",
			body: `  httpNodeCors: { origin: ["https://a.example.com", "https://b.example.com"] },`,
			want: &model.HTTPNodeCorsConfig{Origin: []string{"https://a.example.com", "https://b.example.com"}},
		},
		{
			name: "methods comma string",
			body: `  httpNodeCors: { origin: "*", methods: "GET, PUT" },`,
			want: &model.HTTPNodeCorsConfig{Origin: []string{"*"}, Methods: []string{"GET", "PUT"}},
		},
		{
			name: "methods array",
			body: `  httpNodeCors: { origin: "*", methods: ["GET", "POST"] },`,
			want: &model.HTTPNodeCorsConfig{Origin: []string{"*"}, Methods: []string{"GET", "POST"}},
		},
		{
			name: "credentials true",
			body: `  httpNodeCors: { origin: "https://app.example.com", credentials: true },`,
			want: &model.HTTPNodeCorsConfig{Origin: []string{"https://app.example.com"}, Credentials: true},
		},
		{
			name: "credentials false",
			body: `  httpNodeCors: { origin: "*", credentials: false },`,
			want: &model.HTTPNodeCorsConfig{Origin: []string{"*"}},
		},
		{
			name: "allowed headers array",
			body: `  httpNodeCors: { origin: "*", allowedHeaders: ["Content-Type", "X-Token"] },`,
			want: &model.HTTPNodeCorsConfig{Origin: []string{"*"}, AllowedHeaders: []string{"Content-Type", "X-Token"}},
		},
		{
			name: "allowed headers comma string",
			body: `  httpNodeCors: { origin: "*", allowedHeaders: "Content-Type, X-Token" },`,
			want: &model.HTTPNodeCorsConfig{Origin: []string{"*"}, AllowedHeaders: []string{"Content-Type", "X-Token"}},
		},
		{
			name: "full block",
			body: `  httpNodeCors: {
    origin: ["https://app.example.com"],
    methods: ["GET", "PUT", "POST", "DELETE"],
    credentials: true,
    allowedHeaders: ["X-Token"],
  },`,
			want: &model.HTTPNodeCorsConfig{
				Origin:         []string{"https://app.example.com"},
				Methods:        []string{"GET", "PUT", "POST", "DELETE"},
				Credentials:    true,
				AllowedHeaders: []string{"X-Token"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := "module.exports = {\n" + tc.body + "\n  uiPort: 1880,\n}\n"
			got := parseHTTPNodeCorsBlockFromJS(doc)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseHTTPNodeCorsBlockFromJS() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// TestParseHTTPNodeCorsBlockFromJS_Unsupported locks the read-only contract
// (spec S3): every shape NRCC cannot model comes back as a non-nil config
// flagged Unsupported whose Raw keeps the verbatim block, so the UI can show
// it and the patcher can leave it untouched.
func TestParseHTTPNodeCorsBlockFromJS_Unsupported(t *testing.T) {
	cases := []struct {
		name  string
		block string
	}{
		{"function origin", `  httpNodeCors: { origin: function(req, cb) { cb(null, true); } },`},
		{"arrow function origin", `  httpNodeCors: { origin: (req, cb) => cb(null, true) },`},
		{"regexp constructor", `  httpNodeCors: { origin: new RegExp('.*') },`},
		{"regex literal", `  httpNodeCors: { origin: /example\.com/ },`},
		{"nested object", `  httpNodeCors: { origin: { host: "app.example.com" } },`},
		{"unknown key", `  httpNodeCors: { origin: "*", maxAge: 5 },`},
		{"non-literal value", `  httpNodeCors: { origin: allowedOrigin },`},
		{"empty object", `  httpNodeCors: { },`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := "module.exports = {\n" + tc.block + "\n  uiPort: 1880,\n}\n"
			got := parseHTTPNodeCorsBlockFromJS(doc)
			if got == nil {
				t.Fatalf("parseHTTPNodeCorsBlockFromJS() = nil, want an Unsupported config")
			}
			if !got.Unsupported {
				t.Fatalf("parseHTTPNodeCorsBlockFromJS() = %#v, want Unsupported: true", got)
			}
			if got.Raw != tc.block {
				t.Fatalf("Raw = %q, want the verbatim block %q", got.Raw, tc.block)
			}
		})
	}
}

// TestRenderHTTPNodeCorsBlock_RoundTrip pins spec S3's renderer: parse of the
// rendered block reproduces the config, and rendering that parse again is
// byte-stable.
func TestRenderHTTPNodeCorsBlock_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		cfg  *model.HTTPNodeCorsConfig
	}{
		{"wildcard", &model.HTTPNodeCorsConfig{Origin: []string{"*"}}},
		{"single origin", &model.HTTPNodeCorsConfig{Origin: []string{"https://app.example.com"}}},
		{"multiple origins", &model.HTTPNodeCorsConfig{Origin: []string{"https://a.example.com", "https://b.example.com"}}},
		{"methods", &model.HTTPNodeCorsConfig{Origin: []string{"*"}, Methods: []string{"GET", "PUT"}}},
		{"credentials", &model.HTTPNodeCorsConfig{Origin: []string{"https://app.example.com"}, Credentials: true}},
		{"allowed headers", &model.HTTPNodeCorsConfig{Origin: []string{"*"}, AllowedHeaders: []string{"Content-Type", "X-Token"}}},
		{
			name: "everything",
			cfg: &model.HTTPNodeCorsConfig{
				Origin:         []string{"https://app.example.com"},
				Methods:        []string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE"},
				Credentials:    true,
				AllowedHeaders: []string{"X-Token"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			block := renderHTTPNodeCorsBlock(model.NodeRedConfig{HTTPNodeCors: tc.cfg})
			if block == "" {
				t.Fatal("renderHTTPNodeCorsBlock() = \"\", want a rendered block")
			}
			doc := "module.exports = {\n" + block + "\n  uiPort: 1880,\n}\n"
			parsed := parseHTTPNodeCorsBlockFromJS(doc)
			if parsed == nil || parsed.Unsupported {
				t.Fatalf("parse(render(cfg)) = %#v, want the modelable config", parsed)
			}
			if !reflect.DeepEqual(parsed, tc.cfg) {
				t.Fatalf("parse(render(cfg)) = %#v, want %#v", parsed, tc.cfg)
			}
			if again := renderHTTPNodeCorsBlock(model.NodeRedConfig{HTTPNodeCors: parsed}); again != block {
				t.Fatalf("render(parse(render(cfg))) drifted:\nfirst=%q\nagain=%q", block, again)
			}
		})
	}
}

// TestRenderHTTPNodeCorsBlock_EmptyForNilAndUnsupported pins the renderer's
// "nothing to write" contract: nil config means the key is absent and an
// unsupported block is never rewritten.
func TestRenderHTTPNodeCorsBlock_EmptyForNilAndUnsupported(t *testing.T) {
	if got := renderHTTPNodeCorsBlock(model.NodeRedConfig{}); got != "" {
		t.Fatalf("renderHTTPNodeCorsBlock(nil) = %q, want \"\"", got)
	}
	unsupported := &model.HTTPNodeCorsConfig{Unsupported: true, Raw: "  httpNodeCors: { origin: fn },"}
	if got := renderHTTPNodeCorsBlock(model.NodeRedConfig{HTTPNodeCors: unsupported}); got != "" {
		t.Fatalf("renderHTTPNodeCorsBlock(unsupported) = %q, want \"\"", got)
	}
}

// TestSave_RoundTripsHTTPNodeCorsBlock mirrors the TLS round-trip style: the
// rendered block reaches settings.js and a second Save does not duplicate it.
func TestSave_RoundTripsHTTPNodeCorsBlock(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")

	existing := `module.exports = {
  uiPort: 1880,
  httpAdminRoot: "/",
  httpNodeRoot: "/",
  adminAuth: null,
}
`
	if err := os.WriteFile(settingsPath, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := model.DefaultNodeRedConfig()
	cfg.HTTPNodeCors = &model.HTTPNodeCorsConfig{
		Origin:      []string{"https://app.example.com"},
		Methods:     []string{"GET", "POST"},
		Credentials: true,
	}
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	after := readFileString(t, settingsPath)
	for _, want := range []string{
		"httpNodeCors: {",
		`origin: ["https://app.example.com"],`,
		`methods: ["GET", "POST"],`,
		"credentials: true,",
	} {
		if !strings.Contains(after, want) {
			t.Fatalf("Save dropped %q from settings.js:\n%s", want, after)
		}
	}

	if err := svc.Save(cfg); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	twice := readFileString(t, settingsPath)
	if n := strings.Count(twice, "httpNodeCors:"); n != 1 {
		t.Fatalf("httpNodeCors appears %d times after two saves:\n%s", n, twice)
	}
}

// TestPatchSettingsJS_RemovesHTTPNodeCorsWhenNil pins "clear means delete":
// a nil config drops the whole block without touching its neighbours.
func TestPatchSettingsJS_RemovesHTTPNodeCorsWhenNil(t *testing.T) {
	body := `module.exports = {
  uiPort: 1880,
  httpNodeCors: {
    origin: ["*"],
    methods: ["GET"],
  },
  adminAuth: null,
}
`
	cfg := model.DefaultNodeRedConfig()
	got := patchSettingsJS(body, cfg)
	if strings.Contains(got, "httpNodeCors:") {
		t.Fatalf("patchSettingsJS kept httpNodeCors when cfg.HTTPNodeCors was nil:\n%s", got)
	}
	if !strings.Contains(got, "adminAuth: null,") {
		t.Fatalf("patchSettingsJS dropped an unrelated key:\n%s", got)
	}
}

// TestPatchSettingsJS_LeavesUnsupportedHTTPNodeCorsUntouched pins that a
// hand-written block NRCC cannot model survives a save byte-for-byte,
// including its function body and its original formatting.
func TestPatchSettingsJS_LeavesUnsupportedHTTPNodeCorsUntouched(t *testing.T) {
	block := `  httpNodeCors: {
    origin: function(req, cb) { cb(null, req.headers.origin === "https://app.example.com"); },
    methods: "GET,PUT",
  },`
	body := "module.exports = {\n" + block + "\n  uiPort: 1880,\n  adminAuth: null,\n}\n"
	cfg := model.DefaultNodeRedConfig()
	cfg.HTTPNodeCors = parseHTTPNodeCorsBlockFromJS(body)
	if cfg.HTTPNodeCors == nil || !cfg.HTTPNodeCors.Unsupported {
		t.Fatalf("fixture did not parse as unsupported: %#v", cfg.HTTPNodeCors)
	}

	got := patchSettingsJS(body, cfg)
	if !strings.Contains(got, block) {
		t.Fatalf("patchSettingsJS rewrote the unsupported block:\n%s", got)
	}
	if n := strings.Count(got, "httpNodeCors:"); n != 1 {
		t.Fatalf("httpNodeCors appears %d times, want 1:\n%s", n, got)
	}
}

// TestNodeRED5Catalog_CORSEntry pins the catalog row required by spec S1 and
// the managed-key ownership that lets the block flow through the existing
// apply pipeline (S7).
func TestNodeRED5Catalog_CORSEntry(t *testing.T) {
	var entry *SettingCatalogEntry
	for _, e := range NodeRED5Catalog() {
		if e.Key == "httpNodeCors" {
			e := e
			entry = &e
			break
		}
	}
	if entry == nil {
		t.Fatalf("Node-RED 5 catalog is missing the `httpNodeCors` entry required by #1076")
	}
	want := SettingCatalogEntry{
		Key:             "httpNodeCors",
		Shape:           "http-node-cors",
		Default:         "undefined",
		Validation:      "http-node-cors",
		RestartRequired: true,
		UIEditable:      true,
	}
	if *entry != want {
		t.Fatalf("httpNodeCors catalog entry = %#v, want %#v", *entry, want)
	}
	if !IsManagedSettingKey("httpNodeCors") {
		t.Fatalf("httpNodeCors is not a managed setting key; the apply pipeline would not rewrite it")
	}
	if IsSecretSettingKey("httpNodeCors") {
		t.Fatalf("httpNodeCors must not be a secret setting key (no credentials in the block)")
	}
}
