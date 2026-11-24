package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// envGetAs calls GET /api/env with the given claims (nil means no claims).
func envGetAs(t *testing.T, claims *model.Claims) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/env", nil)
	if claims != nil {
		req = req.WithContext(context.WithValue(req.Context(), middleware.CtxKeyUser, claims))
	}
	return req
}

// Issue #1022: GET /api/env masks plain values for every non-admin caller,
// the same way GET /api/config does, and returns them in clear for admins.
func TestGetEnv_MasksValuesForNonAdmin(t *testing.T) {
	configSvc := service.NewIsolatedConfigService(t.TempDir())
	cfg, err := configSvc.Get()
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	cfg.EnvVars = []model.EnvVar{{Key: "API_URL", Value: "https://internal.example", Type: "string"}}
	if err := configSvc.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	h := NewEnvHandler(service.NewEnvService(configSvc), t.TempDir())

	tests := []struct {
		name   string
		claims *model.Claims
		want   string
	}{
		{"viewer sees masked value", &model.Claims{Username: "v", Role: model.RoleViewer}, "********"},
		{"missing claims fail closed", nil, "********"},
		{"admin sees value", &model.Claims{Username: "a", Role: model.RoleAdmin}, "https://internal.example"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.GetEnv(w, envGetAs(t, tt.claims))
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
			}
			var resp struct {
				Data []model.EnvVar `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(resp.Data) != 1 || resp.Data[0].Key != "API_URL" {
				t.Fatalf("unexpected vars: %+v", resp.Data)
			}
			if got := resp.Data[0].Value; got != tt.want {
				t.Errorf("value = %q, want %q", got, tt.want)
			}
		})
	}

	// The masked read must not rewrite the stored value.
	stored, err := configSvc.Get()
	if err != nil {
		t.Fatalf("re-read config: %v", err)
	}
	if stored.EnvVars[0].Value != "https://internal.example" {
		t.Errorf("stored value changed to %q", stored.EnvVars[0].Value)
	}
}
