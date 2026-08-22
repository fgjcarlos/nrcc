package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
)

// newAuthzTestServer builds a server together with the auth service so the test
// can mint admin and viewer tokens for the same signing secret.
func newAuthzTestServer(t *testing.T) (*Server, *service.AuthService) {
	t.Helper()
	dir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](dir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](dir + "/sessions.json")
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)
	srv := NewServerWithConfig(authSvc, Config{DataDir: dir, CORS: middleware.CORSConfig{}, HTTPLogger: discardHTTPLogger()})
	return srv, authSvc
}

func tokenForRole(t *testing.T, authSvc *service.AuthService, id string, role model.UserRole) string {
	t.Helper()
	now := time.Now().Format(time.RFC3339)
	token, err := authSvc.GenerateToken(&model.CCUser{
		ID:           id,
		Username:     id,
		PasswordHash: "hash",
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		t.Fatalf("GenerateToken(%s): %v", role, err)
	}
	return token
}

var mutatingRoutes = []struct{ method, path string }{
	{http.MethodPost, "/api/backups/"},
	{http.MethodPost, "/api/backups/config"},
	{http.MethodDelete, "/api/backups/some-id"},
	// #674: backup download exposes cc-users.json (bcrypt hashes) and
	// flows_cred.json — viewer must not reach it.
	{http.MethodGet, "/api/backups/some-id/download"},
	{http.MethodPost, "/api/backups/some-id/restore"},
	// #675: provider snapshots leak provider name + remote repo layout
	// (fingerprintable); the sibling GetBackupProvider is already gated
	// for the same reason.
	{http.MethodGet, "/api/backups/provider/snapshots"},
	{http.MethodPost, "/api/scheduler/config"},
	{http.MethodPatch, "/api/storage/retention"},
	{http.MethodPost, "/api/env/"},
	{http.MethodDelete, "/api/env/SOME_KEY"},
	// #673: GET /api/env/dotenv returns raw .env unfiltered, defeating
	// the masking on GET /api/env. Admin-only.
	{http.MethodGet, "/api/env/dotenv"},
	{http.MethodPut, "/api/env/dotenv"},
	{http.MethodPost, "/api/flows/versions"},
	{http.MethodPost, "/api/flows/versions/v1/revert"},
	{http.MethodPost, "/api/libraries/install"},
	{http.MethodDelete, "/api/libraries/some-pkg"},
	{http.MethodPost, "/api/updates/apply"},
	{http.MethodPost, "/api/files/upload"},
	{http.MethodDelete, "/api/files/some-file"},
}

// TestAuthz_ViewerForbiddenOnMutatingRoutes is the #274 regression: a viewer
// token must be rejected with 403 on every state-mutating endpoint.
func TestAuthz_ViewerForbiddenOnMutatingRoutes(t *testing.T) {
	srv, authSvc := newAuthzTestServer(t)
	viewerToken := tokenForRole(t, authSvc, "viewer", model.RoleViewer)

	for _, route := range mutatingRoutes {
		req := httptest.NewRequest(route.method, route.path, nil)
		req.Header.Set("Authorization", "Bearer "+viewerToken)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("viewer %s %s: expected 403, got %d", route.method, route.path, rec.Code)
		}
	}
}

// TestAuthz_AdminPassesAdminGate verifies the admin role is not blocked by the
// RequireAdmin gate (it may still get 4xx/5xx from the handler for an empty
// body, but never 403 from the authorization layer).
func TestAuthz_AdminPassesAdminGate(t *testing.T) {
	srv, authSvc := newAuthzTestServer(t)
	adminToken := tokenForRole(t, authSvc, "admin", model.RoleAdmin)

	for _, route := range mutatingRoutes {
		req := httptest.NewRequest(route.method, route.path, nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code == http.StatusForbidden {
			t.Errorf("admin %s %s: must not be blocked by RequireAdmin, got 403", route.method, route.path)
		}
	}
}

// TestAuthz_ViewerCanReadGetEndpoints ensures the admin gate did not regress
// read access — viewers must still reach GET endpoints.
func TestAuthz_ViewerCanReadGetEndpoints(t *testing.T) {
	srv, authSvc := newAuthzTestServer(t)
	viewerToken := tokenForRole(t, authSvc, "viewer", model.RoleViewer)

	reads := []string{
		"/api/backups/",
		"/api/env/",
		"/api/flows/",
		"/api/libraries/",
		"/api/updates/status",
	}
	for _, path := range reads {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+viewerToken)
		rec := httptest.NewRecorder()

		srv.ServeHTTP(rec, req)

		if rec.Code == http.StatusForbidden || rec.Code == http.StatusUnauthorized {
			t.Errorf("viewer GET %s: read access regressed, got %d", path, rec.Code)
		}
	}
}

func TestSecurityPostureAuthzBoundary(t *testing.T) {
	srv, authSvc := newAuthzTestServer(t)
	defer srv.Shutdown()

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "unauthenticated", wantStatus: http.StatusUnauthorized},
		{name: "viewer", token: tokenForRole(t, authSvc, "viewer-posture", model.RoleViewer), wantStatus: http.StatusForbidden},
		{name: "admin", token: tokenForRole(t, authSvc, "admin-posture", model.RoleAdmin), wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/system/security-posture", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if tt.wantStatus != http.StatusOK {
				if _, exposed := body["data"]; exposed {
					t.Fatalf("unauthorized response exposed posture data: %v", body)
				}
				return
			}
			data, ok := body["data"].(map[string]any)
			if !ok || data["activeRefreshSessions"] != float64(0) {
				t.Fatalf("admin response missing canonical zero counts: %v", body)
			}
		})
	}
}
