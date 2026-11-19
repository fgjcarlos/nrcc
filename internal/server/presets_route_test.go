package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TestPresetsRoutes_Mounted pins the curated-preset API (#764 slice 3) to
// the router. Before it was mounted, both paths fell through to the SPA
// handler and returned 404 API_ROUTE_NOT_FOUND. Both routes are admin-only,
// mirroring /api/settings/raw: the preview consumes and returns settings.js
// source.
func TestPresetsRoutes_Mounted(t *testing.T) {
	srv, authSvc := newAuthzTestServer(t)
	admin := tokenForRole(t, authSvc, "admin", model.RoleAdmin)
	viewer := tokenForRole(t, authSvc, "viewer", model.RoleViewer)
	previewBody := `{"content":"module.exports = {\n}\n"}`

	cases := []struct {
		name     string
		method   string
		path     string
		body     string
		token    string
		wantCode int
		wantErr  string // expected error.code; empty for success
		wantBody string // substring expected in a success body
	}{
		{"admin lists catalogue", http.MethodGet, "/api/presets", "", admin, http.StatusOK, "", `"id":"https-tls-preset"`},
		{"unauthenticated list", http.MethodGet, "/api/presets", "", "", http.StatusUnauthorized, "UNAUTHORIZED", ""},
		{"viewer list", http.MethodGet, "/api/presets", "", viewer, http.StatusForbidden, "FORBIDDEN", ""},
		{"admin preview unknown id", http.MethodPost, "/api/presets/no-such-preset/preview", previewBody, admin, http.StatusNotFound, "PRESET_NOT_REGISTERED", ""},
		{"admin preview malformed body", http.MethodPost, "/api/presets/https-tls-preset/preview", "{", admin, http.StatusBadRequest, "INVALID_REQUEST", ""},
		{"unauthenticated preview", http.MethodPost, "/api/presets/https-tls-preset/preview", previewBody, "", http.StatusUnauthorized, "UNAUTHORIZED", ""},
		{"viewer preview", http.MethodPost, "/api/presets/https-tls-preset/preview", previewBody, viewer, http.StatusForbidden, "FORBIDDEN", ""},
		{"admin preview valid redacts", http.MethodPost, "/api/presets/https-tls-preset/preview", `{"content":"module.exports = {\n  credentialSecret: \"hunter2\",\n  https: {\n    passphrase: \"hunter3\",\n  },\n}\n","values":{"key":"/data/key.pem","cert":"/data/cert.pem"}}`, admin, http.StatusOK, "", `credentialSecret: [redacted]`},
		{"admin preview https with paths", http.MethodPost, "/api/presets/https-tls-preset/preview", `{"content":"module.exports = {\n  uiPort: 1880,\n}\n","values":{"key":"/data/key.pem","cert":"/data/cert.pem"}}`, admin, http.StatusOK, "", `readFileSync(\"/data/cert.pem\")`},
		{"admin preview https without paths", http.MethodPost, "/api/presets/https-tls-preset/preview", previewBody, admin, http.StatusUnprocessableEntity, "PRESET_NEEDS_INPUT", ""},
		{"admin preview empty body", http.MethodPost, "/api/presets/https-tls-preset/preview", "", admin, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body"},
		{"admin preview body over 1 MiB within 2 MiB", http.MethodPost, "/api/presets/https-tls-preset/preview", `{"content":"` + strings.Repeat("a", 3<<19) + `"}`, admin, http.StatusUnprocessableEntity, "SOURCE_NOT_EXPORTS", ""},
		{"admin preview body over 2 MiB", http.MethodPost, "/api/presets/https-tls-preset/preview", `{"content":"` + strings.Repeat("a", 2<<20) + `"}`, admin, http.StatusRequestEntityTooLarge, "", "request body too large"},
		{"admin preview encoded traversal id", http.MethodPost, "/api/presets/..%2F..%2Fetc%2Fpasswd/preview", previewBody, admin, http.StatusNotFound, "PRESET_NOT_REGISTERED", ""},
		{"admin preview empty content", http.MethodPost, "/api/presets/https-tls-preset/preview", `{"content":""}`, admin, http.StatusUnprocessableEntity, "SOURCE_NOT_EXPORTS", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tc.wantCode, rec.Body.String())
			}
			if tc.wantBody != "" && !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Errorf("body missing %s: %s", tc.wantBody, rec.Body.String())
			}
			if tc.wantErr == "" {
				return
			}
			var env model.ApiErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env.Error == nil {
				t.Fatalf("decode error envelope: err=%v body=%s", err, rec.Body.String())
			}
			if env.Error.Code != tc.wantErr {
				t.Errorf("error.code = %q, want %q", env.Error.Code, tc.wantErr)
			}
		})
	}
}
