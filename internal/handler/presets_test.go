package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test slice 3 of issue #764 — backend handler for the curated
// presets catalogue. The handler serves two endpoints that the
// AdvancedSettings UI consumes:
//
//   - GET  /api/presets                  — list curated presets
//   - POST /api/presets/{id}/preview     — preview a preset apply
//
// Apply-to-disk is intentionally NOT in this handler: the operator
// reviews the redacted preview and then confirms by writing through
// the existing /api/settings/raw endpoint, so the settings.js write
// path stays single-flight and audited (slice 2 of #758).

func TestPresetsHandler_ListReturnsCorePresets(t *testing.T) {
	h := NewPresetsHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/presets", nil)
	h.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("List status: got %d want %d", w.Code, http.StatusOK)
	}
	var body struct {
		Success bool                       `json:"success"`
		Data    []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.Success {
		t.Fatalf("expected success=true, got %+v", body)
	}
	ids := map[string]bool{}
	for _, p := range body.Data {
		id, _ := p["id"].(string)
		ids[id] = true
	}
	for _, want := range []string{
		"https-tls-preset",
		"functionGlobalContext-strict",
		"functionGlobalContext-relaxed",
		"logging-callback-middleware",
	} {
		if !ids[want] {
			t.Errorf("List response missing core preset %q (got %v)", want, ids)
		}
	}
}

// Test fixture: settings.js with a passphrase-bearing https block,
// formatted so source_patch.go's findModuleExportsClosingBrace can
// find the closing brace on its own line.
const presetPreviewFixture = `{"content":"module.exports = {\n  https: {\n    key: 'k',\n    cert: 'c',\n    passphrase: 'super-secret',\n  },\n}\n"}`

func TestPresetsHandler_PreviewRedactsSecrets(t *testing.T) {
	h := NewPresetsHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/presets/https-tls-preset/preview", strings.NewReader(presetPreviewFixture))
	r.Header.Set("Content-Type", "application/json")
	h.Preview(w, r, "https-tls-preset")

	if w.Code != http.StatusOK {
		t.Fatalf("Preview status: got %d want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "super-secret") {
		t.Errorf("preview leaked passphrase: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "[redacted]") {
		t.Errorf("preview should contain [redacted] marker, got: %s", w.Body.String())
	}
}

func TestPresetsHandler_PreviewUnknownPreset(t *testing.T) {
	h := NewPresetsHandler()
	body := `{"content":"module.exports = {}"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/presets/no-such-preset/preview", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	h.Preview(w, r, "no-such-preset")

	if w.Code != http.StatusNotFound {
		t.Fatalf("Preview unknown status: got %d want %d", w.Code, http.StatusNotFound)
	}
}

func TestPresetsHandler_PreviewMalformedJSON(t *testing.T) {
	h := NewPresetsHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/presets/https-tls-preset/preview", strings.NewReader("{not-json"))
	r.Header.Set("Content-Type", "application/json")
	h.Preview(w, r, "https-tls-preset")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Preview malformed status: got %d want %d", w.Code, http.StatusBadRequest)
	}
}
