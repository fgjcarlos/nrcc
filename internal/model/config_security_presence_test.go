package model

import (
	"encoding/json"
	"testing"
)

// TestNodeRedConfig_SecuritySurfacePresence covers the issue #881 rule: a
// payload that omits an authentication surface must be distinguishable from a
// payload that clears it, and a config built in Go stays authoritative.
func TestNodeRedConfig_SecuritySurfacePresence(t *testing.T) {
	tests := []struct {
		name              string
		body              string
		adminAuthCarried  bool
		nodeAuthCarried   bool
		staticAuthCarried bool
	}{
		{
			name:              "omitted surfaces are not carried",
			body:              `{"uiPort":1880}`,
			adminAuthCarried:  false,
			nodeAuthCarried:   false,
			staticAuthCarried: false,
		},
		{
			name:              "an explicit null is carried",
			body:              `{"uiPort":1880,"adminAuth":null,"httpNodeAuth":null,"httpStaticAuth":null}`,
			adminAuthCarried:  true,
			nodeAuthCarried:   true,
			staticAuthCarried: true,
		},
		{
			name:              "a populated surface is carried",
			body:              `{"uiPort":1880,"adminAuth":{"type":"credentials","users":[]},"httpNodeAuth":{"user":"u","pass":"p"}}`,
			adminAuthCarried:  true,
			nodeAuthCarried:   true,
			staticAuthCarried: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg NodeRedConfig
			if err := json.Unmarshal([]byte(tt.body), &cfg); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got := cfg.AdminAuthCarried(); got != tt.adminAuthCarried {
				t.Errorf("AdminAuthCarried() = %v, want %v", got, tt.adminAuthCarried)
			}
			if got := cfg.HTTPNodeAuthCarried(); got != tt.nodeAuthCarried {
				t.Errorf("HTTPNodeAuthCarried() = %v, want %v", got, tt.nodeAuthCarried)
			}
			if got := cfg.HTTPStaticAuthCarried(); got != tt.staticAuthCarried {
				t.Errorf("HTTPStaticAuthCarried() = %v, want %v", got, tt.staticAuthCarried)
			}
		})
	}
}

// TestNodeRedConfig_ProgrammaticConfigIsAuthoritative proves a config built
// in Go is treated as a complete model: the presence rules only apply to
// payloads decoded from JSON.
func TestNodeRedConfig_ProgrammaticConfigIsAuthoritative(t *testing.T) {
	cfg := NodeRedConfig{
		AdminAuth:      &AdminAuth{Type: "credentials"},
		HTTPNodeAuth:   &HTTPBasicAuth{User: "u", Pass: "p"},
		HTTPStaticAuth: &HTTPBasicAuth{User: "u", Pass: "p"},
	}

	if !cfg.AdminAuthCarried() {
		t.Error("a programmatic config must carry adminAuth")
	}
	if !cfg.HTTPNodeAuthCarried() {
		t.Error("a programmatic config must carry httpNodeAuth")
	}
	if !cfg.HTTPStaticAuthCarried() {
		t.Error("a programmatic config must carry httpStaticAuth")
	}
}

// TestNodeRedConfig_MarshalKeepsSurfacesOutOfTheStore proves the presence
// bookkeeping never leaks into the persisted JSON.
func TestNodeRedConfig_MarshalKeepsSurfacesOutOfTheStore(t *testing.T) {
	var cfg NodeRedConfig
	if err := json.Unmarshal([]byte(`{"uiPort":1880,"adminAuth":{"type":"credentials"}}`), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal marshalled config: %v", err)
	}
	for _, forbidden := range []string{"decodedFromJSON", "adminAuthAbsent", "httpNodeAuthAbsent", "httpStaticAuthAbsent"} {
		if _, ok := raw[forbidden]; ok {
			t.Errorf("the persisted config must not expose %q", forbidden)
		}
	}
}
