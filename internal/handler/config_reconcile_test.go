package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// seedStructuredMirror persists the structured representation of the
// seeded settings.js in config.json, the state a real deployment has once
// the mirror exists. Without the mirror the structured read falls back to
// the source and the defect stays invisible.
func seedStructuredMirror(t *testing.T, f *applyHandlerFixture) {
	t.Helper()
	seed, err := f.svc.Get()
	if err != nil {
		t.Fatalf("seed read failed: %v", err)
	}
	if _, err := f.svc.Update(func(current *model.NodeRedConfig) error {
		*current = seed
		return nil
	}); err != nil {
		t.Fatalf("seed mirror failed: %v", err)
	}
}

// TestApplyRaw_ReconcilesStructuredState is the reproduction test from
// issue #880: a successful raw apply updates settings.js, and the
// structured configuration must report the applied values afterwards.
func TestApplyRaw_ReconcilesStructuredState(t *testing.T) {
	f := newApplyHandlerFixture(t)
	seedStructuredMirror(t, f)

	content := `module.exports = {
  uiPort: 7777,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
};
`
	body, _ := json.Marshal(ApplyRawRequest{Content: content, ExpectedRevision: f.liveRevision(t).Fingerprint})
	r, w := adminRequest(t, http.MethodPost, "/api/config/apply/raw", body)
	f.handler.ApplyRaw(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	cfg, err := f.svc.Get()
	if err != nil {
		t.Fatalf("structured read failed: %v", err)
	}
	if cfg.UIPort != 7777 {
		t.Fatalf("expected the structured read to report uiPort 7777, got %d", cfg.UIPort)
	}
}

// TestApplyRaw_UnrelatedVisualEditKeepsRawChange covers the second half
// of the issue: after the raw apply the next visual save must not revert
// the raw change.
func TestApplyRaw_UnrelatedVisualEditKeepsRawChange(t *testing.T) {
	f := newApplyHandlerFixture(t)
	seedStructuredMirror(t, f)

	content := `module.exports = {
  uiPort: 7777,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
};
`
	body, _ := json.Marshal(ApplyRawRequest{Content: content, ExpectedRevision: f.liveRevision(t).Fingerprint})
	r, w := adminRequest(t, http.MethodPost, "/api/config/apply/raw", body)
	f.handler.ApplyRaw(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// The visual editor loads the structured configuration and sends an
	// unrelated change back.
	cfg, err := f.svc.Get()
	if err != nil {
		t.Fatalf("structured read failed: %v", err)
	}
	cfg.HTTPAdminRoot = "/nrcc"
	if err := f.svc.Save(cfg); err != nil {
		t.Fatalf("structured save failed: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(f.dir, "settings.js"))
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}
	if !strings.Contains(string(raw), "uiPort: 7777") {
		t.Fatalf("the visual save reverted the raw change:\n%s", raw)
	}
}

// TestApplyRaw_RejectedReconciliationLeavesBothRepresentationsUnchanged
// asserts that a raw apply the structured mirror rejects has no durable
// effect: the source keeps its previous bytes and the mirror keeps its
// previous values.
func TestApplyRaw_RejectedReconciliationLeavesBothRepresentationsUnchanged(t *testing.T) {
	f := newApplyHandlerFixture(t)
	seedStructuredMirror(t, f)

	before, err := os.ReadFile(f.settingsPath)
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}

	// An out-of-range port is a valid document for the parser but not for
	// the configuration, so the reconciliation rejects it.
	content := `module.exports = {
  uiPort: 99999,
  httpAdminRoot: '/',
  httpNodeRoot: '/',
};
`
	body, _ := json.Marshal(ApplyRawRequest{Content: content, ExpectedRevision: f.liveRevision(t).Fingerprint})
	r, w := adminRequest(t, http.MethodPost, "/api/config/apply/raw", body)
	f.handler.ApplyRaw(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	after, err := os.ReadFile(f.settingsPath)
	if err != nil {
		t.Fatalf("read settings.js: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("the rejected apply changed the source:\n%s", after)
	}
	cfg, err := f.svc.Get()
	if err != nil {
		t.Fatalf("structured read failed: %v", err)
	}
	if cfg.UIPort != 1880 {
		t.Fatalf("the rejected apply changed the structured read: uiPort %d", cfg.UIPort)
	}
}
