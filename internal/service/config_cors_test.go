package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TestValidateHTTPNodeCors pins the S4 rules and their exact messages. A nil
// or unsupported config is never rejected: it is either absent or read-only.
func TestValidateHTTPNodeCors(t *testing.T) {
	cases := []struct {
		name string
		cors *model.HTTPNodeCorsConfig
		want string
	}{
		{"nil", nil, ""},
		{"unsupported", &model.HTTPNodeCorsConfig{Unsupported: true, Raw: "  httpNodeCors: { origin: fn },"}, ""},
		{"wildcard", &model.HTTPNodeCorsConfig{Origin: []string{"*"}}, ""},
		{"single origin", &model.HTTPNodeCorsConfig{Origin: []string{"https://app.example.com"}}, ""},
		{"two origins", &model.HTTPNodeCorsConfig{Origin: []string{"http://localhost:3000", "https://app.example.com"}}, ""},
		{"credentials with concrete origin", &model.HTTPNodeCorsConfig{Origin: []string{"https://app.example.com"}, Credentials: true}, ""},
		{"empty methods", &model.HTTPNodeCorsConfig{Origin: []string{"*"}}, ""},
		{"all methods", &model.HTTPNodeCorsConfig{
			Origin:  []string{"*"},
			Methods: []string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE"},
		}, ""},
		{"allowed headers", &model.HTTPNodeCorsConfig{Origin: []string{"*"}, AllowedHeaders: []string{"Content-Type", "X-Token"}}, ""},

		{"empty origin", &model.HTTPNodeCorsConfig{}, "httpNodeCors origin must not be empty"},
		{"origin with path", &model.HTTPNodeCorsConfig{Origin: []string{"https://app.example.com/editor"}},
			`httpNodeCors origin "https://app.example.com/editor" must be "*" or an http(s) URL without a path`},
		{"origin with trailing slash", &model.HTTPNodeCorsConfig{Origin: []string{"https://app.example.com/"}},
			`httpNodeCors origin "https://app.example.com/" must be "*" or an http(s) URL without a path`},
		{"origin without scheme", &model.HTTPNodeCorsConfig{Origin: []string{"app.example.com"}},
			`httpNodeCors origin "app.example.com" must be "*" or an http(s) URL without a path`},
		{"origin with query", &model.HTTPNodeCorsConfig{Origin: []string{"https://app.example.com?x=1"}},
			`httpNodeCors origin "https://app.example.com?x=1" must be "*" or an http(s) URL without a path`},
		{"origin non http scheme", &model.HTTPNodeCorsConfig{Origin: []string{"ftp://app.example.com"}},
			`httpNodeCors origin "ftp://app.example.com" must be "*" or an http(s) URL without a path`},
		{"origin empty string", &model.HTTPNodeCorsConfig{Origin: []string{""}},
			`httpNodeCors origin "" must be "*" or an http(s) URL without a path`},
		{"wildcard with credentials", &model.HTTPNodeCorsConfig{Origin: []string{"*"}, Credentials: true},
			`httpNodeCors origin "*" cannot be combined with credentials`},
		{"lowercase method", &model.HTTPNodeCorsConfig{Origin: []string{"*"}, Methods: []string{"get"}},
			`httpNodeCors method "get" is not one of GET, HEAD, PUT, PATCH, POST, DELETE`},
		{"unknown method", &model.HTTPNodeCorsConfig{Origin: []string{"*"}, Methods: []string{"GET", "OPTIONS"}},
			`httpNodeCors method "OPTIONS" is not one of GET, HEAD, PUT, PATCH, POST, DELETE`},
		{"empty header name", &model.HTTPNodeCorsConfig{Origin: []string{"*"}, AllowedHeaders: []string{"Content-Type", ""}},
			"httpNodeCors allowedHeaders must not contain an empty header name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateHTTPNodeCors(tc.cors)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("validateHTTPNodeCors() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateHTTPNodeCors() = nil, want %q", tc.want)
			}
			if err.Error() != tc.want {
				t.Fatalf("validateHTTPNodeCors() = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

// TestValidate_WiresHTTPNodeCors proves the rule is reachable through the
// service entry point the handler uses, not just as a standalone helper.
func TestValidate_WiresHTTPNodeCors(t *testing.T) {
	svc := NewIsolatedConfigService(t.TempDir())
	cfg := model.DefaultNodeRedConfig()
	cfg.HTTPNodeCors = &model.HTTPNodeCorsConfig{Origin: []string{"*"}, Credentials: true}

	err := svc.Validate(cfg)
	if err == nil {
		t.Fatal("Validate() = nil, want the httpNodeCors wildcard+credentials error")
	}
	want := `httpNodeCors origin "*" cannot be combined with credentials`
	if err.Error() != want {
		t.Fatalf("Validate() = %q, want %q", err.Error(), want)
	}
}

// TestSave_RejectsInvalidHTTPNodeCors proves a rejected CORS config writes
// neither the structured store nor settings.js.
func TestSave_RejectsInvalidHTTPNodeCors(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)

	cfg := model.DefaultNodeRedConfig()
	cfg.HTTPNodeCors = &model.HTTPNodeCorsConfig{Origin: []string{"*"}, Credentials: true}

	err := svc.Save(cfg)
	if err == nil {
		t.Fatal("Save() = nil, want the httpNodeCors validation error")
	}
	want := `httpNodeCors origin "*" cannot be combined with credentials`
	if err.Error() != want {
		t.Fatalf("Save() = %q, want %q", err.Error(), want)
	}
	for _, name := range []string{"config.json", "settings.js"} {
		if _, statErr := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(statErr) {
			t.Fatalf("Save wrote %s despite the validation error (stat err: %v)", name, statErr)
		}
	}
}
