package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
)

// newAuthSvcForPosture builds an AuthService backed by temporary stores
// so the posture handler can resolve users + sessions without touching
// the real data dir.
func newAuthSvcForPosture(t *testing.T) *service.AuthService {
	t.Helper()
	dir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](dir + "/cc-users.json")
	sessStore := store.NewJSONStore[model.RefreshSessions](dir + "/refresh-sessions.json")
	return service.NewAuthService("test-secret", userStore, sessStore)
}

func newMfaSvcForPosture(t *testing.T, auth *service.AuthService) *service.MfaService {
	t.Helper()
	dir := t.TempDir()
	return service.NewMfaService(dir, auth)
}

func decodePosture(t *testing.T, body []byte) SecurityPostureResponse {
	t.Helper()
	var resp struct {
		Success   bool                     `json:"success"`
		Data      SecurityPostureResponse  `json:"data"`
		Timestamp string                   `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode response: %v: body=%s", err, body)
	}
	if !resp.Success {
		t.Fatalf("expected success=true, body=%s", body)
	}
	return resp.Data
}

// TestSecurityPosture_Healthy exercises the common-path case: encryption
// key present, downloads admin-gated, no active sessions, one admin
// without MFA enrolled. Backups admin-gated stays true (it's hardcoded
// to track the router-level gate).
func TestSecurityPosture_Healthy(t *testing.T) {
	auth := newAuthSvcForPosture(t)
	if err := auth.CreateUser(&model.CCUser{ID: "u1", Username: "admin", Role: model.RoleAdmin, PasswordHash: "x"}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	mfa := newMfaSvcForPosture(t, auth)

	h := NewSystemHandler()
	h.SetAuthService(auth)
	h.SetMfaService(mfa)

	// EnvService needs a config svc + encryption key. We don't need to
	// exercise the env surface — only EncryptionKeyConfigured. Build a
	// bare EnvService directly with the key set.
	configSvc := service.NewConfigServiceWithHost(t.TempDir(), service.NewHostService(t.TempDir()))
	env := service.NewEnvService(configSvc, "this-is-a-real-key")
	h.SetEnvService(env)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/system/security-posture", nil)
	h.GetSecurityPosture(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d (body=%s)", w.Code, w.Body.String())
	}
	got := decodePosture(t, w.Body.Bytes())
	if !got.EncryptionKeyConfigured {
		t.Error("encryptionKeyConfigured: want true (key was set)")
	}
	if !got.BackupAccessAdminGated {
		t.Error("backupAccessAdminGated: want true (hardcoded)")
	}
	if got.ActiveRefreshSessions != 0 {
		t.Errorf("activeRefreshSessions: want 0, got %d", got.ActiveRefreshSessions)
	}
	if got.TotalAdmins != 1 {
		t.Errorf("totalAdmins: want 1, got %d", got.TotalAdmins)
	}
	if got.MfaEnrolledAdmins != 0 {
		t.Errorf("mfaEnrolledAdmins: want 0 (no enrollment), got %d", got.MfaEnrolledAdmins)
	}
}

// TestSecurityPosture_NoEncryptionKey covers the degraded state the
// card exists to surface: key absent, so Encrypted env vars would be
// written in clear.
func TestSecurityPosture_NoEncryptionKey(t *testing.T) {
	auth := newAuthSvcForPosture(t)
	mfa := newMfaSvcForPosture(t, auth)

	h := NewSystemHandler()
	h.SetAuthService(auth)
	h.SetMfaService(mfa)

	configSvc := service.NewConfigServiceWithHost(t.TempDir(), service.NewHostService(t.TempDir()))
	env := service.NewEnvService(configSvc) // no key
	h.SetEnvService(env)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/system/security-posture", nil)
	h.GetSecurityPosture(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}
	got := decodePosture(t, w.Body.Bytes())
	if got.EncryptionKeyConfigured {
		t.Error("encryptionKeyConfigured: want false (key not set)")
	}
}

// TestSecurityPosture_NilDependencies verifies the handler stays
// servable when only a subset of deps is wired (early bootstrap path).
// All count fields must return 0 and booleans false.
func TestSecurityPosture_NilDependencies(t *testing.T) {
	h := NewSystemHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/system/security-posture", nil)
	h.GetSecurityPosture(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}
	got := decodePosture(t, w.Body.Bytes())
	if got.EncryptionKeyConfigured {
		t.Error("encryptionKeyConfigured: want false (no envSvc)")
	}
	if !got.BackupAccessAdminGated {
		t.Error("backupAccessAdminGated: stays true (hardcoded)")
	}
	if got.ActiveRefreshSessions != 0 || got.TotalAdmins != 0 || got.MfaEnrolledAdmins != 0 {
		t.Errorf("counts must be 0 with nil deps, got %+v", got)
	}
}

// TestSecurityPosture_ActiveSessions prunes expired/revoked sessions
// from the chip.
func TestSecurityPosture_ActiveSessions(t *testing.T) {
	auth := newAuthSvcForPosture(t)
	// Seed three sessions: active, expired, revoked. Only the active
	// one should be counted.
	if err := auth.CreateUser(&model.CCUser{ID: "u1", Username: "u", Role: model.RoleAdmin, PasswordHash: "x"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	mustSession := func(s model.RefreshSession) {
		if err := auth.CreateRefreshSessionForTest(s); err != nil {
			t.Fatalf("seed session: %v", err)
		}
	}
	mustSession(model.RefreshSession{ID: "a", UserID: "u1", ExpiresAt: 9_999_999_999})
	mustSession(model.RefreshSession{ID: "b", UserID: "u1", ExpiresAt: 1, Revoked: false}) // expired
	mustSession(model.RefreshSession{ID: "c", UserID: "u1", ExpiresAt: 9_999_999_999, Revoked: true})

	h := NewSystemHandler()
	h.SetAuthService(auth)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/system/security-posture", nil)
	h.GetSecurityPosture(w, r)
	got := decodePosture(t, w.Body.Bytes())
	if got.ActiveRefreshSessions != 1 {
		t.Errorf("activeRefreshSessions: want 1 (only 'a'), got %d", got.ActiveRefreshSessions)
	}
}