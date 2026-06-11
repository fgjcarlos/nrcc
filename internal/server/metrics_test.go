package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](dir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](dir + "/sessions.json")
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)
	return NewServerWithConfig(authSvc, dir, middleware.CORSConfig{})
}

// TestMetricsEndpoint_Returns200 verifies that GET /metrics returns HTTP 200.
func TestMetricsEndpoint_Returns200(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestMetricsEndpoint_ContainsPrometheusFormat verifies the response uses the
// Prometheus text exposition format (presence of Go runtime metrics is always guaranteed).
func TestMetricsEndpoint_ContainsPrometheusFormat(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	body := rec.Body.String()
	// Go collector metrics are always present after registration.
	if !strings.Contains(body, "go_goroutines") {
		t.Fatalf("/metrics response missing Prometheus format content; body:\n%s", body)
	}
}

// TestMetricsEndpoint_ContainsNodeRedRuntimeMetrics verifies that the process
// collector metrics registered by PR 1 are present (nrcc_nodered_running gauge).
// These are always exported because the ProcessCollector is a Describe-based collector.
func TestMetricsEndpoint_ContainsNodeRedRuntimeMetrics(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "nrcc_nodered_running") {
		t.Fatalf("/metrics response missing 'nrcc_nodered_running'; got:\n%s", body)
	}
}

// TestMetricsEndpoint_ContainsLoginAttemptAfterActivity verifies that
// nrcc_login_attempts_total appears after a login attempt is recorded.
// This proves the wiring between AuthHandler and MetricsCollector is correct.
func TestMetricsEndpoint_ContainsLoginAttemptAfterActivity(t *testing.T) {
	srv := newTestServer(t)

	// Trigger a login attempt (will fail — no users — but metric will be recorded).
	loginBody := strings.NewReader(`{"username":"nobody","password":"x"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody)
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	srv.ServeHTTP(loginRec, loginReq)
	// Login must fail (no users exist).
	if loginRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown user, got %d", loginRec.Code)
	}

	// Now scrape /metrics — the counter must appear.
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "nrcc_login_attempts_total") {
		t.Fatalf("/metrics missing 'nrcc_login_attempts_total' after login activity; body:\n%s", body)
	}
}

// TestMetricsEndpoint_IsPublic verifies that /metrics is accessible without auth
// (no Authorization header required).
func TestMetricsEndpoint_IsPublic(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	// Deliberately omit Authorization header
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	// Should NOT return 401 Unauthorized
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("/metrics must be public (no auth required), got 401")
	}
}
