package server

import (
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
	srv := NewServerWithConfig(authSvc, dir, middleware.CORSConfig{})
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
	{http.MethodPost, "/api/backups/some-id/restore"},
	{http.MethodPost, "/api/scheduler/config"},
	{http.MethodPatch, "/api/storage/retention"},
	{http.MethodPost, "/api/env/"},
	{http.MethodDelete, "/api/env/SOME_KEY"},
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
