package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// Issue #878 — "prevent JSON commits on stale revision rejection (P1)".
//
// The audited finding was that a structured apply committed config.json
// before the expectedRevision precondition was enforced, so a rejected
// request still changed the structured state while leaving settings.js
// untouched.
//
// The transaction guards that close the finding live in the apply
// coordinator (revision precondition → commit → backup → write →
// restore). The coordinator already rejects a stale revision before the
// commit, so the JSON-commit half of the finding is closed on this
// baseline. What was still open is the visual save: POST /api/config
// generated a fresh server-side expectation instead of carrying the
// revision the operator actually read, so a stale browser session
// silently overwrote newer state.
//
// The tests below pin both halves:
//   - the release-audit reproduction from the issue, verbatim;
//   - the two-session scenario on the visual save surface;
//   - the store/file invariants on every rejected transaction;
//   - the concurrent-environment-edit invariant.

// revisionSaveFixture wires a ConfigHandler with an apply coordinator over
// an isolated config service, mirroring the production wiring (see
// internal/server/server.go).
type revisionSaveFixture struct {
	dir          string
	svc          *service.ConfigService
	handler      *ConfigHandler
	settingsPath string
	configPath   string
	recorder     *applyHandlerRecorder
}

func newRevisionSaveFixture(t *testing.T) *revisionSaveFixture {
	t.Helper()
	dir := t.TempDir()
	svc := service.NewIsolatedConfigService(dir)
	// Seed the JSON store through the production save path so it carries
	// the derived metadata (settingsPath / settingsSource) a real
	// deployment stores, then restore the settings.js fixture content the
	// handler tests use.
	if err := svc.Save(svc.GetDefault()); err != nil {
		t.Fatalf("seed config store: %v", err)
	}
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJSDoc(t)), 0o600); err != nil {
		t.Fatalf("seed settings.js: %v", err)
	}
	recorder := &applyHandlerRecorder{}
	apply := service.NewApplyService(svc, recorder.Log)
	coordinator := service.NewApplyCoordinator(apply)
	handler := NewConfigHandler(svc)
	handler.SetApplyCoordinator(coordinator)
	return &revisionSaveFixture{
		dir:          dir,
		svc:          svc,
		handler:      handler,
		settingsPath: settingsPath,
		configPath:   filepath.Join(dir, "config.json"),
		recorder:     recorder,
	}
}

// seedStore publishes a configuration into the JSON store so the tests can
// compare the structured state before and after a rejected transaction.
// The settings.js file is left untouched — the store and the file are the
// two independent representations the issue is about.
//
// The derived metadata (settingsPath / settingsSource) is preserved: it is
// owned by the host detection, not by the operator, and the store commit
// re-derives it on every save.
func (f *revisionSaveFixture) seedStore(t *testing.T, cfg model.NodeRedConfig) {
	t.Helper()
	if _, err := f.svc.Update(func(current *model.NodeRedConfig) error {
		settingsPath, settingsSource := current.SettingsPath, current.SettingsSource
		*current = cfg
		current.SettingsPath = settingsPath
		current.SettingsSource = settingsSource
		return nil
	}); err != nil {
		t.Fatalf("seed config store: %v", err)
	}
}

// configBytes reads the JSON store representation.
func (f *revisionSaveFixture) configBytes(t *testing.T) string {
	t.Helper()
	// #nosec G304 -- configPath is derived from t.TempDir(), not user input.
	data, err := os.ReadFile(f.configPath)
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}
	return string(data)
}

// settingsBytes reads the settings.js representation.
func (f *revisionSaveFixture) settingsBytes(t *testing.T) string {
	t.Helper()
	// #nosec G304 -- settingsPath is derived from t.TempDir(), not user input.
	data, err := os.ReadFile(f.settingsPath)
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}
	return string(data)
}

// save posts a structured configuration to POST /api/config.
func (f *revisionSaveFixture) save(t *testing.T, payload map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req, w := adminRequest(t, http.MethodPost, "/api/config", body)
	f.handler.SaveConfig(w, req)
	return w
}

// configPayload is the structured body the visual save sends.
func configPayload(overrides map[string]interface{}) map[string]interface{} {
	payload := map[string]interface{}{
		"uiPort":          7777,
		"httpAdminRoot":   "/",
		"httpNodeRoot":    "/",
		"projectsEnabled": false,
		"logging":         map[string]interface{}{},
	}
	for key, value := range overrides {
		payload[key] = value
	}
	return payload
}

// TestReleaseAuditStaleApplyMustNotCommitJSON is the regression
// reproduction from issue #878, verbatim. It asserts that a rejected
// transaction leaves both representations (config.json and settings.js)
// unchanged.
func TestReleaseAuditStaleApplyMustNotCommitJSON(t *testing.T) {
	f := newApplyHandlerFixture(t)
	before, err := f.svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	// #nosec G304 -- settingsPath is derived from t.TempDir(), not user input.
	sourceBefore, err := os.ReadFile(f.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]interface{}{
		"uiPort": 7777, "httpAdminRoot": "/", "httpNodeRoot": "/",
		"projectsEnabled": false, "logging": map[string]interface{}{},
		"expectedRevision": strings.Repeat("a", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	req, w := adminRequest(t, http.MethodPost, "/api/config/apply", body)
	f.handler.ApplyStructured(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	after, err := f.svc.Get()
	if err != nil {
		t.Fatal(err)
	}
	// #nosec G304 -- settingsPath is derived from t.TempDir(), not user input.
	sourceAfter, err := os.ReadFile(f.settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(sourceBefore) != string(sourceAfter) {
		t.Error("rejected transaction changed settings.js")
	}
	if before.UIPort != after.UIPort {
		t.Errorf("rejected transaction changed config.json: uiPort %d -> %d", before.UIPort, after.UIPort)
	}
}

// TestConfigHandler_SaveConfig_StaleRevisionRejectedKeepsBothRepresentations
// is the two-browser-session scenario from the acceptance criteria:
//
//	session A reads the configuration (revision R0)
//	session B saves a newer configuration (revision R1)
//	session A saves its own edit with R0 → rejected, nothing durable changed
//
// Without the client-supplied precondition the third step silently
// overwrote session B's configuration.
func TestConfigHandler_SaveConfig_StaleRevisionRejectedKeepsBothRepresentations(t *testing.T) {
	f := newRevisionSaveFixture(t)
	f.seedStore(t, f.svc.GetDefault())

	// Session A reads the live document.
	sessionA, err := f.svc.GetRawSettings()
	if err != nil {
		t.Fatalf("session A read: %v", err)
	}

	// Session B saves a newer configuration.
	if w := f.save(t, configPayload(nil)); w.Code != http.StatusOK {
		t.Fatalf("session B status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	configAfterB := f.configBytes(t)
	settingsAfterB := f.settingsBytes(t)

	// Session A saves its stale edit with the revision it read.
	w := f.save(t, configPayload(map[string]interface{}{
		"uiPort":           9999,
		"expectedRevision": sessionA.Revision.Fingerprint,
	}))
	if w.Code != http.StatusConflict {
		t.Fatalf("session A status = %d, want 409; body = %s", w.Code, w.Body.String())
	}
	liveFP, providedFP, ok := decodeRevisionConflict(t, w.Body.Bytes())
	if !ok {
		t.Fatalf("response is not a SETTINGS_REVISION_CONFLICT envelope: %s", w.Body.String())
	}
	if providedFP != sessionA.Revision.Fingerprint {
		t.Errorf("envelope providedRevision = %q, want %q", providedFP, sessionA.Revision.Fingerprint)
	}
	sessionB, err := f.svc.GetRawSettings()
	if err != nil {
		t.Fatalf("session B read: %v", err)
	}
	if liveFP != sessionB.Revision.Fingerprint {
		t.Errorf("envelope liveRevision = %q, want %q", liveFP, sessionB.Revision.Fingerprint)
	}

	// Neither representation may change.
	if got := f.configBytes(t); got != configAfterB {
		t.Errorf("rejected save changed config.json:\n before: %s\n after:  %s", configAfterB, got)
	}
	if got := f.settingsBytes(t); got != settingsAfterB {
		t.Errorf("rejected save changed settings.js:\n before: %s\n after:  %s", settingsAfterB, got)
	}
	stored, err := f.svc.Get()
	if err != nil {
		t.Fatalf("read stored config: %v", err)
	}
	if stored.UIPort != 7777 {
		t.Errorf("stored uiPort = %d, want session B's 7777", stored.UIPort)
	}
}

// TestConfigHandler_SaveConfig_AcceptedRevisionCommits asserts the happy
// path still works when the client sends the revision it read.
func TestConfigHandler_SaveConfig_AcceptedRevisionCommits(t *testing.T) {
	f := newRevisionSaveFixture(t)
	f.seedStore(t, f.svc.GetDefault())

	live, err := f.svc.GetRawSettings()
	if err != nil {
		t.Fatalf("read live settings: %v", err)
	}
	w := f.save(t, configPayload(map[string]interface{}{
		"expectedRevision": live.Revision.Fingerprint,
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	stored, err := f.svc.Get()
	if err != nil {
		t.Fatalf("read stored config: %v", err)
	}
	if stored.UIPort != 7777 {
		t.Errorf("stored uiPort = %d, want 7777", stored.UIPort)
	}
	if !strings.Contains(f.settingsBytes(t), "uiPort: 7777") {
		t.Errorf("settings.js was not updated: %s", f.settingsBytes(t))
	}
}
