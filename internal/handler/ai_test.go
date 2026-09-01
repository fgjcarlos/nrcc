package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

func newAIConfigHandler(t *testing.T, probe service.AIProviderProbe) (*AIHandler, *service.AIConfigService) {
	t.Helper()
	svc := service.NewAIConfigService(t.TempDir(), "encryption-key", service.WithAIProviderProbe(probe))
	return NewAIHandler(nil, svc), svc
}

func aiRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	return req.WithContext(context.WithValue(req.Context(), middleware.CtxKeyUser, &model.Claims{Username: "admin", Role: model.RoleAdmin}))
}

func TestAIHandlerConfigAndStatus(t *testing.T) {
	h, svc := newAIConfigHandler(t, func(context.Context, service.AIConfig) error { return nil })
	if err := svc.Save(service.AIConfig{Enabled: true, Provider: "offline", Model: "local"}); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	for _, tt := range []struct {
		name   string
		method string
		path   string
		body   string
		call   http.HandlerFunc
		want   string
	}{
		{"get config redacts key", http.MethodGet, "/api/ai/config", "", h.GetConfig, `"apiKeyConfigured":false`},
		{"get status is ready", http.MethodGet, "/api/ai/status", "", h.GetStatus, `"status":"ready"`},
		{"save remote config", http.MethodPut, "/api/ai/config", `{"enabled":true,"provider":"openai","endpoint":"https://api.example.test","model":"model","apiKey":"secret"}`, h.PutConfig, `"apiKeyConfigured":true`},
		{"test configured provider", http.MethodPost, "/api/ai/config/test", "", h.TestConfig, `"status":"ready"`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.call(w, aiRequest(tt.method, tt.path, tt.body))
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
			if !bytes.Contains(w.Body.Bytes(), []byte(tt.want)) || bytes.Contains(w.Body.Bytes(), []byte("secret")) {
				t.Fatalf("unexpected response: %s", w.Body.String())
			}
		})
	}
}

func TestAIHandlerMapsConfigErrors(t *testing.T) {
	tests := []struct {
		name       string
		probe      service.AIProviderProbe
		seed       *service.AIConfig
		call       func(*AIHandler, *httptest.ResponseRecorder)
		wantStatus int
		wantCode   string
	}{
		{
			name: "invalid provider",
			call: func(h *AIHandler, w *httptest.ResponseRecorder) {
				h.PutConfig(w, aiRequest(http.MethodPut, "/api/ai/config", `{"enabled":true,"provider":"unknown"}`))
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "AI_INVALID_PROVIDER",
		},
		{
			name: "invalid endpoint",
			call: func(h *AIHandler, w *httptest.ResponseRecorder) {
				h.PutConfig(w, aiRequest(http.MethodPut, "/api/ai/config", `{"enabled":true,"provider":"openai","endpoint":"http://invalid.example.test","model":"model","apiKey":"secret"}`))
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "AI_INVALID_ENDPOINT",
		},
		{
			name: "missing key",
			seed: &service.AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model"},
			call: func(h *AIHandler, w *httptest.ResponseRecorder) {
				h.TestConfig(w, aiRequest(http.MethodPost, "/api/ai/config/test", ""))
			},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "AI_INCOMPLETE",
		},
		{
			name:  "provider timeout",
			probe: func(context.Context, service.AIConfig) error { return context.DeadlineExceeded },
			seed:  &service.AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model", APIKey: "secret"},
			call: func(h *AIHandler, w *httptest.ResponseRecorder) {
				h.TestConfig(w, aiRequest(http.MethodPost, "/api/ai/config/test", ""))
			},
			wantStatus: http.StatusGatewayTimeout,
			wantCode:   "AI_PROBE_TIMEOUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, svc := newAIConfigHandler(t, tt.probe)
			if tt.seed != nil {
				if err := svc.Save(*tt.seed); err != nil {
					t.Fatalf("seed config: %v", err)
				}
			}
			w := httptest.NewRecorder()
			tt.call(h, w)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
			var resp model.ApiErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if resp.Error == nil || resp.Error.Code != tt.wantCode {
				t.Fatalf("error = %#v, want %s", resp.Error, tt.wantCode)
			}
			if bytes.Contains(w.Body.Bytes(), []byte("secret")) {
				t.Fatalf("error response leaked secret: %s", w.Body.String())
			}
		})
	}
}

func TestAIHandlerAnalyzeFlowUsesPersistedProviderConfiguration(t *testing.T) {
	h, svc := newAIConfigHandler(t, nil)
	if err := svc.Save(service.AIConfig{Enabled: true, Provider: "offline", Model: "local"}); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	w := httptest.NewRecorder()
	h.PostAnalyzeFlow(w, aiRequest(http.MethodPost, "/api/ai/analyze/flow", `{"action":"explain","flow":{"id":"flow-1","nodes":[]}}`))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"provider":"offline"`)) {
		t.Fatalf("response did not use persisted provider: %s", w.Body.String())
	}
}

func TestAIHandlerAnalyzeFlowRejectsNonReadyPersistedStatus(t *testing.T) {
	tests := []struct {
		name       string
		cfg        service.AIConfig
		connection string
		wantStatus int
	}{
		{"disabled", service.AIConfig{Provider: "offline"}, "", http.StatusServiceUnavailable},
		{"incomplete", service.AIConfig{Enabled: true, Provider: "openai", Model: "model"}, "", http.StatusServiceUnavailable},
		{"testing", service.AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model", APIKey: "secret"}, "testing", http.StatusServiceUnavailable},
		{"unreachable", service.AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model", APIKey: "secret"}, "unreachable", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			probe := func(context.Context, service.AIConfig) error {
				if tt.connection != "testing" {
					return errors.New("connection refused")
				}
				close(started)
				<-release
				return errors.New("connection refused")
			}
			h, svc := newAIConfigHandler(t, probe)
			if err := svc.Save(tt.cfg); err != nil {
				t.Fatalf("seed config: %v", err)
			}
			if tt.connection != "" {
				if tt.connection == "testing" {
					testDone := make(chan error, 1)
					go func() { _, err := svc.Test(context.Background()); testDone <- err }()
					<-started
					defer func() {
						close(release)
						<-testDone
					}()
				} else if _, err := svc.Test(context.Background()); err == nil {
					t.Fatal("Test() error = nil, want unavailable provider")
				}
			}
			w := httptest.NewRecorder()
			h.PostAnalyzeFlow(w, aiRequest(http.MethodPost, "/api/ai/analyze/flow", `{"action":"explain","flow":{"id":"flow-1","nodes":[]}}`))
			if w.Code != tt.wantStatus || !bytes.Contains(w.Body.Bytes(), []byte(`"code":"AI_NOT_READY"`)) {
				t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
			}
		})
	}
}
