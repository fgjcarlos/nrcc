package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyRawHandler_RejectsInvalidJavaScriptWithoutMutation(t *testing.T) {
	for _, tc := range []struct{ name, content, message string }{
		{"syntax", "module.exports = { uiPort: 1880, broken: };", "syntax error"},
		{"runtime", "throw new Error('inspection failed'); module.exports = {};", "sandbox runtime error"},
		{"undefined variable", "module.exports = { uiPort: unknownPort };", "sandbox runtime error"},
		{"syntax after host access", "const os = require('os'); module.exports = { broken: };", "syntax error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newApplyHandlerFixture(t)
			configPath := filepath.Join(f.dir, "config.json")
			configBefore := []byte("{\"uiPort\":1880}\n")
			if err := os.WriteFile(configPath, configBefore, 0o600); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(f.settingsPath)
			if err != nil {
				t.Fatal(err)
			}
			revision := f.liveRevision(t)
			body, err := json.Marshal(ApplyRawRequest{Content: tc.content, ExpectedRevision: revision.Fingerprint})
			if err != nil {
				t.Fatal(err)
			}
			req, w := adminRequest(t, http.MethodPost, "/api/config/apply/raw", body)
			f.handler.ApplyRaw(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400; body = %s", w.Code, w.Body.String())
			}
			var envelope struct {
				Error struct{ Code, Message string }
			}
			if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Error.Code != "VALIDATION_ERROR" || !strings.Contains(envelope.Error.Message, tc.message) {
				t.Errorf("unexpected error envelope: %s", w.Body.String())
			}
			for path, expected := range map[string][]byte{f.settingsPath: before, configPath: configBefore} {
				// #nosec G304 -- path is one of the isolated fixture's settings/config files.
				after, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(after, expected) {
					t.Errorf("rejected content changed %s", path)
				}
			}
			if got := f.liveRevision(t); got.Fingerprint != revision.Fingerprint || got.Algorithm != revision.Algorithm {
				t.Errorf("revision changed: got %+v, want %+v", got, revision)
			}
			if _, err := os.Stat(filepath.Join(f.dir, "backups")); !os.IsNotExist(err) {
				t.Errorf("rejection created backup state: %v", err)
			}
			var sawFailure bool
			for _, event := range f.recorder.events {
				if event.Action == "apply.write" || event.Action == "apply.backup" || event.Action == "apply.success" {
					t.Errorf("rejection advanced beyond validation: %+v", event)
				}
				if event.Action == "apply.failure" && event.ApplyStage == "validate" {
					sawFailure = true
				}
			}
			if !sawFailure {
				t.Error("missing validate-stage failure audit event")
			}
		})
	}
}

func TestApplyRawHandler_PreservesSourceManagedDynamicSettings(t *testing.T) {
	for _, content := range []string{
		"const os = require('os'); module.exports = { uiPort: 1880, hostname: os.hostname() };",
		"module.exports = { uiPort: process.env.PORT || 1880 };",
		"module.exports = { uiPort: 1880, adminAuth: null };",
		"module.exports = { uiPort: 1880, adminAuth: { type: 'custom', authenticate: function() { return null; } } };",
	} {
		f := newApplyHandlerFixture(t)
		body, err := json.Marshal(ApplyRawRequest{Content: content, ExpectedRevision: f.liveRevision(t).Fingerprint})
		if err != nil {
			t.Fatal(err)
		}
		req, w := adminRequest(t, http.MethodPost, "/api/config/apply/raw", body)
		f.handler.ApplyRaw(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("valid source rejected: %s; status %d: %s", content, w.Code, w.Body.String())
		}
		after, err := os.ReadFile(f.settingsPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != content {
			t.Fatalf("dynamic source was altered: %q", after)
		}
	}
}
