package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// applyHandlerFixture wires the slice C ConfigApplyHandler
// with an isolated config service + apply coordinator. The
// recorder captures the audit events the handler + coordinator
// emit so tests can assert the failure_stage envelope.
type applyHandlerFixture struct {
	dir         string
	svc         *service.ConfigService
	apply       *service.ApplyService
	coordinator *service.ApplyCoordinator
	handler     *ConfigApplyHandler
	settingsPath string
	recorder    *applyHandlerRecorder
}

type applyHandlerRecorder struct {
	mu     sync.Mutex
	events []handlerRecordedAudit
}

type handlerRecordedAudit struct {
	Action       string
	Result       string
	FailureStage string
}

func (r *applyHandlerRecorder) Log(_ *http.Request, _, action, _, result string, meta map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, handlerRecordedAudit{
		Action:       action,
		Result:       result,
		FailureStage: meta["failure_stage"],
	})
}

func newApplyHandlerFixture(t *testing.T) *applyHandlerFixture {
	t.Helper()
	dir := t.TempDir()
	svc := service.NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJSDoc(t)), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	rec := &applyHandlerRecorder{}
	apply := service.NewApplyService(svc, rec.Log)
	coord := service.NewApplyCoordinator(apply)
	handler := NewConfigApplyHandler(svc, coord)
	handler.SetAuditService(nil) // audit is via the recorder, not the in-memory audit.Service
	return &applyHandlerFixture{
		dir:          dir,
		svc:          svc,
		apply:        apply,
		coordinator:  coord,
		handler:      handler,
		settingsPath: settingsPath,
		recorder:     rec,
	}
}

// validSettingsJSDoc is a minimal but parseable settings.js
// for the handler tests. We mirror the slice A
// validSettingsJS helper but keep this file self-contained so
// the handler package can compile independently of the
// service-test helper.
func validSettingsJSDoc(t *testing.T) string {
	t.Helper()
	return `module.exports = {
  uiPort: 1880,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
  credentialSecret: 'initial-secret-value',
  adminAuth: null,
};
`
}

// adminRequest builds an httptest.NewRequest with admin
// claims injected into the context so the handler's
// authorisation check passes. Returns the request + a
// recorder to capture the response.
func adminRequest(t *testing.T, method, target string, body []byte) (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, target, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	ctx := context.WithValue(r.Context(), middleware.CtxKeyUser, &model.Claims{
		Username: "admin",
		Role:     model.RoleAdmin,
	})
	return r.WithContext(ctx), httptest.NewRecorder()
}

// decodeRevisionConflict decodes the 409 envelope the slice C
// handlers emit so tests can assert the live/provided fields.
func decodeRevisionConflict(t *testing.T, body []byte) (liveRevision, providedRevision string, ok bool) {
	t.Helper()
	var env struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Data *struct {
			LiveRevision     model.SourceRevision `json:"liveRevision"`
			ProvidedRevision string               `json:"providedRevision"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if env.Error == nil || env.Error.Code != "SETTINGS_REVISION_CONFLICT" {
		return "", "", false
	}
	if env.Data == nil {
		return "", "", false
	}
	return env.Data.LiveRevision.Fingerprint, env.Data.ProvidedRevision, true
}

// liveRevision reads the live document's revision through the
// config service for the handler tests.
func (f *applyHandlerFixture) liveRevision(t *testing.T) model.SourceRevision {
	t.Helper()
	doc, err := f.svc.GetRawSettings()
	if err != nil {
		t.Fatalf("GetRawSettings: %v", err)
	}
	return doc.Revision
}

// TestApplyStructuredHandler_HappyPath drives
// POST /api/config/apply with a valid payload and asserts a
// 200 + audit envelope.
func TestApplyStructuredHandler_HappyPath(t *testing.T) {
	f := newApplyHandlerFixture(t)

	preApplyRevision := f.liveRevision(t).Fingerprint

	payload := map[string]interface{}{
		"uiPort":          7777,
		"httpAdminRoot":   "/",
		"httpNodeRoot":    "/",
		"projectsEnabled": false,
		"logging":         map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)
	req, w := adminRequest(t, http.MethodPost, "/api/config/apply", body)

	f.handler.ApplyStructured(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var envelope model.ApiResponse[ApplyStructuredResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	resp := envelope.Data
	if resp.Configuration.UIPort != 7777 {
		t.Errorf("Configuration.UIPort = %d, want 7777", resp.Configuration.UIPort)
	}
	if resp.Configuration.Port != 7777 {
		t.Errorf("Configuration.Port = %d, want 7777 (UIPort fallback)", resp.Configuration.Port)
	}
	if resp.Document.Revision.Fingerprint == "" {
		t.Error("Document.Revision.Fingerprint is empty")
	}
	if resp.Document.Revision.Fingerprint == preApplyRevision {
		t.Errorf("Document.Revision.Fingerprint did not advance after apply: %s",
			resp.Document.Revision.Fingerprint)
	}
	// Audit envelope: the apply coordinator must have
	// emitted apply.success with failure_stage=ok. Other
	// events (apply.start etc.) are emitted by slice A's
	// ApplyService.
	if len(f.recorder.events) == 0 {
		t.Fatal("no audit events recorded")
	}
	var sawSuccess bool
	for _, ev := range f.recorder.events {
		if ev.Action == "apply.success" && ev.FailureStage == "ok" {
			sawSuccess = true
		}
	}
	if !sawSuccess {
		t.Errorf("did not see apply.success with failure_stage=ok; events: %+v", f.recorder.events)
	}
}

// TestApplyStructuredHandler_StaleRevisionRejected drives the
// structured apply with a stale expectedRevision and asserts
// the 409 SETTINGS_REVISION_CONFLICT envelope.
func TestApplyStructuredHandler_StaleRevisionRejected(t *testing.T) {
	f := newApplyHandlerFixture(t)

	stale := f.liveRevision(t)
	stale.Fingerprint = strings.Repeat("a", 64)

	payload := map[string]interface{}{
		"uiPort":           7777,
		"httpAdminRoot":    "/",
		"httpNodeRoot":     "/",
		"projectsEnabled":  false,
		"logging":          map[string]interface{}{},
		"expectedRevision": stale.Fingerprint,
	}
	body, _ := json.Marshal(payload)
	req, w := adminRequest(t, http.MethodPost, "/api/config/apply", body)

	f.handler.ApplyStructured(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body = %s", w.Code, w.Body.String())
	}
	liveFP, providedFP, ok := decodeRevisionConflict(t, w.Body.Bytes())
	if !ok {
		t.Fatalf("response did not decode as SETTINGS_REVISION_CONFLICT envelope: %s", w.Body.String())
	}
	if liveFP != f.liveRevision(t).Fingerprint {
		t.Errorf("envelope liveRevision = %q, want %q", liveFP, f.liveRevision(t).Fingerprint)
	}
	if providedFP != stale.Fingerprint {
		t.Errorf("envelope providedRevision = %q, want %q", providedFP, stale.Fingerprint)
	}
}

// TestApplyStructuredHandler_RequiresAdmin ensures the handler
// rejects non-admin requests. The middleware enforces this on
// the route; the handler itself trusts the claim-in-context
// contract documented on SaveConfig.
func TestApplyStructuredHandler_RequiresAdminClaims(t *testing.T) {
	f := newApplyHandlerFixture(t)
	req := httptest.NewRequest(http.MethodPost, "/api/config/apply", nil)
	w := httptest.NewRecorder()
	f.handler.ApplyStructured(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (no claims)", w.Code)
	}
}

// TestApplyRawHandler_HappyPath covers POST /api/config/apply/raw.
func TestApplyRawHandler_HappyPath(t *testing.T) {
	f := newApplyHandlerFixture(t)
	live := f.liveRevision(t)
	const content = `module.exports = { uiPort: 1880, httpAdminRoot: '/', httpNodeRoot: '/' };` + "\n"

	body, _ := json.Marshal(ApplyRawRequest{
		Content:          content,
		ExpectedRevision: live.Fingerprint,
	})
	req, w := adminRequest(t, http.MethodPost, "/api/config/apply/raw", body)
	f.handler.ApplyRaw(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var envelope model.ApiResponse[ApplyRawResponse]
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	resp := envelope.Data
	if resp.Document.Path != f.settingsPath {
		t.Errorf("Document.Path = %q, want %q", resp.Document.Path, f.settingsPath)
	}
	if resp.Document.Revision.Fingerprint == "" {
		t.Error("Document.Revision.Fingerprint is empty after raw apply")
	}
}

// TestApplyRawHandler_StaleRevisionRejected covers the raw
// path's 409 SETTINGS_REVISION_CONFLICT envelope.
func TestApplyRawHandler_StaleRevisionRejected(t *testing.T) {
	f := newApplyHandlerFixture(t)
	stale := f.liveRevision(t)
	stale.Fingerprint = strings.Repeat("b", 64)

	body, _ := json.Marshal(ApplyRawRequest{
		Content:          `module.exports = { uiPort: 1880 };` + "\n",
		ExpectedRevision: stale.Fingerprint,
	})
	req, w := adminRequest(t, http.MethodPost, "/api/config/apply/raw", body)
	f.handler.ApplyRaw(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body = %s", w.Code, w.Body.String())
	}
	liveFP, providedFP, ok := decodeRevisionConflict(t, w.Body.Bytes())
	if !ok {
		t.Fatalf("response did not decode as SETTINGS_REVISION_CONFLICT envelope: %s", w.Body.String())
	}
	if liveFP != f.liveRevision(t).Fingerprint {
		t.Errorf("envelope liveRevision = %q, want %q", liveFP, f.liveRevision(t).Fingerprint)
	}
	if providedFP != stale.Fingerprint {
		t.Errorf("envelope providedRevision = %q, want %q", providedFP, stale.Fingerprint)
	}
}

// TestApplyRawHandler_EmptyContentRejected covers the boundary
// check: an empty content payload is a 400 INVALID_REQUEST
// before any apply work happens.
func TestApplyRawHandler_EmptyContentRejected(t *testing.T) {
	f := newApplyHandlerFixture(t)
	body, _ := json.Marshal(ApplyRawRequest{Content: ""})
	req, w := adminRequest(t, http.MethodPost, "/api/config/apply/raw", body)
	f.handler.ApplyRaw(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

// TestConfigHandler_SaveConfig_DelegatesToApplyCoordinator
// exercises the slice C delegation on the existing
// POST /api/config endpoint: when an ApplyCoordinator is
// wired, SaveConfig routes through it and the settings.js
// write goes through the apply transaction.
func TestConfigHandler_SaveConfig_DelegatesToApplyCoordinator(t *testing.T) {
	dir := t.TempDir()
	svc := service.NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJSDoc(t)), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	rec := &applyHandlerRecorder{}
	apply := service.NewApplyService(svc, rec.Log)
	coord := service.NewApplyCoordinator(apply)

	h := NewConfigHandler(svc)
	h.SetApplyCoordinator(coord)

	payload := map[string]interface{}{
		"uiPort":          7777,
		"httpAdminRoot":   "/",
		"httpNodeRoot":    "/",
		"projectsEnabled": false,
		"logging":         map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)
	req, w := adminRequest(t, http.MethodPost, "/api/config", body)
	h.SaveConfig(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	// The apply transaction must have emitted
	// apply.success — the legacy path emits nothing of the
	// kind.
	var sawSuccess bool
	for _, ev := range rec.events {
		if ev.Action == "apply.success" && ev.FailureStage == "ok" {
			sawSuccess = true
		}
	}
	if !sawSuccess {
		t.Errorf("expected apply.success in audit events; got: %+v", rec.events)
	}

	// On-disk file reflects the new uiPort (via the apply
	// transaction's atomic write).
	// #nosec G304 -- settingsPath is derived from t.TempDir(), not user input.
	got, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}
	if !strings.Contains(string(got), "uiPort: 7777") {
		t.Errorf("settings.js was not updated via the apply transaction: %s", got)
	}
}

// TestSettingsHandler_SaveRaw_DelegatesToApplyCoordinator
// mirrors the structured test for the raw surface.
func TestSettingsHandler_SaveRaw_DelegatesToApplyCoordinator(t *testing.T) {
	dir := t.TempDir()
	svc := service.NewIsolatedConfigService(dir)
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJSDoc(t)), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	rec := &applyHandlerRecorder{}
	apply := service.NewApplyService(svc, rec.Log)
	coord := service.NewApplyCoordinator(apply)

	h := NewSettingsHandler(svc)
	h.SetApplyCoordinator(coord)

	doc, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("seed GetRawSettings: %v", err)
	}

	body, _ := json.Marshal(RawSettingsRequest{
		Content:          `module.exports = { uiPort: 9999, httpAdminRoot: '/', httpNodeRoot: '/' };` + "\n",
		ExpectedRevision: doc.Revision.Fingerprint,
	})
	req, w := adminRequest(t, http.MethodPost, "/api/settings/raw", body)
	h.SaveRaw(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var sawSuccess bool
	for _, ev := range rec.events {
		if ev.Action == "apply.success" && ev.FailureStage == "ok" {
			sawSuccess = true
		}
	}
	if !sawSuccess {
		t.Errorf("expected apply.success in audit events; got: %+v", rec.events)
	}
	// #nosec G304 -- settingsPath is derived from t.TempDir(), not user input.
	got, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}
	if !strings.Contains(string(got), "uiPort: 9999") {
		t.Errorf("settings.js was not updated via the apply transaction: %s", got)
	}
}
