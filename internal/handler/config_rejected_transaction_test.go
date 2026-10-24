package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// Issue #878 — the invariants of a rejected transaction.
//
// Every rejected transaction must leave the two representations — the
// JSON store (config.json) and the settings file (settings.js) — exactly
// as they were. The scenarios below cover the four rejection stages of
// the apply pipeline (conflict, validation, backup, write) plus the
// concurrent-environment-edit invariant.

// TestConfigHandler_SaveConfig_RejectedTransactionKeepsBothRepresentations
// pins the store/file invariants for every rejected transaction: a
// conflict, a validation failure, a backup failure and a write failure
// must all leave config.json and settings.js exactly as they were.
func TestConfigHandler_SaveConfig_RejectedTransactionKeepsBothRepresentations(t *testing.T) {
	cases := []struct {
		name       string
		wantStatus int
		wantCode   string
		prepare    func(t *testing.T, f *revisionSaveFixture) map[string]interface{}
	}{
		{
			name:       "stale revision",
			wantStatus: http.StatusConflict,
			wantCode:   "SETTINGS_REVISION_CONFLICT",
			prepare: func(t *testing.T, f *revisionSaveFixture) map[string]interface{} {
				return configPayload(map[string]interface{}{
					"expectedRevision": strings.Repeat("b", 64),
				})
			},
		},
		{
			name:       "validation failure",
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
			prepare: func(t *testing.T, f *revisionSaveFixture) map[string]interface{} {
				return configPayload(map[string]interface{}{"uiPort": 70000})
			},
		},
		{
			name:       "backup failure",
			wantStatus: http.StatusInternalServerError,
			wantCode:   "SETTINGS_BACKUP_FAILED",
			prepare: func(t *testing.T, f *revisionSaveFixture) map[string]interface{} {
				// A regular file where the backup directory belongs makes
				// the backup stage fail with ENOTDIR for every uid.
				if err := os.WriteFile(filepath.Join(f.dir, "backups"), []byte("not a directory"), 0o600); err != nil {
					t.Fatalf("block the backup directory: %v", err)
				}
				return configPayload(nil)
			},
		},
		{
			name:       "write failure",
			wantStatus: http.StatusInternalServerError,
			wantCode:   "SETTINGS_WRITE_ERROR",
			prepare: func(t *testing.T, f *revisionSaveFixture) map[string]interface{} {
				// A symlink in place of settings.js is rejected by the
				// atomic-write helper before the rename, so the write
				// stage fails while the file itself stays intact.
				target := filepath.Join(f.dir, "settings.target.js")
				if err := os.Rename(f.settingsPath, target); err != nil {
					t.Fatalf("move settings.js aside: %v", err)
				}
				if err := os.Symlink(target, f.settingsPath); err != nil {
					t.Fatalf("symlink settings.js: %v", err)
				}
				return configPayload(nil)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newRevisionSaveFixture(t)
			f.seedStore(t, f.svc.GetDefault())

			payload := tc.prepare(t, f)
			configBefore := f.configBytes(t)
			settingsBefore := f.settingsBytes(t)

			w := f.save(t, payload)
			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", w.Code, tc.wantStatus, w.Body.String())
			}
			var envelope struct {
				Error *struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode envelope: %v", err)
			}
			if envelope.Error == nil || envelope.Error.Code != tc.wantCode {
				t.Errorf("error code = %+v, want %s; body = %s", envelope.Error, tc.wantCode, w.Body.String())
			}
			if got := f.configBytes(t); got != configBefore {
				t.Errorf("rejected transaction changed config.json:\n before: %s\n after:  %s", configBefore, got)
			}
			if got := f.settingsBytes(t); got != settingsBefore {
				t.Errorf("rejected transaction changed settings.js:\n before: %s\n after:  %s", settingsBefore, got)
			}
			for _, event := range f.recorder.events {
				if event.Action == "apply.success" {
					t.Errorf("rejected transaction emitted apply.success: %+v", event)
				}
			}
			// The rejection must stay in the audit trail.
			failures := 0
			for _, event := range f.recorder.events {
				if event.Action == "apply.failure" {
					failures++
				}
			}
			if failures == 0 {
				t.Errorf("rejected transaction emitted no apply.failure event: %+v", f.recorder.events)
			}
		})
	}
}

// TestConfigHandler_SaveConfig_LegacyPathHonoursExpectedRevision covers the
// handler without an apply coordinator. The legacy path has no transaction
// machinery, so the handler enforces the precondition itself — a stale
// save must still be rejected instead of overwriting newer state.
func TestConfigHandler_SaveConfig_LegacyPathHonoursExpectedRevision(t *testing.T) {
	dir := t.TempDir()
	svc := service.NewIsolatedConfigService(dir)
	if err := svc.Save(svc.GetDefault()); err != nil {
		t.Fatalf("seed config store: %v", err)
	}
	settingsPath := filepath.Join(dir, "settings.js")
	if err := os.WriteFile(settingsPath, []byte(validSettingsJSDoc(t)), 0o600); err != nil {
		t.Fatalf("seed settings.js: %v", err)
	}
	handler := NewConfigHandler(svc) // no apply coordinator: the legacy path

	live, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("read live settings: %v", err)
	}
	// #nosec G304 -- config.json lives in the test-owned t.TempDir().
	configBefore, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}

	body, err := json.Marshal(configPayload(map[string]interface{}{
		"expectedRevision": strings.Repeat("c", 64),
	}))
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req, w := adminRequest(t, http.MethodPost, "/api/config", body)
	handler.SaveConfig(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body = %s", w.Code, w.Body.String())
	}
	// #nosec G304 -- config.json lives in the test-owned t.TempDir().
	configAfter, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}
	if string(configAfter) != string(configBefore) {
		t.Errorf("rejected legacy save changed config.json:\n before: %s\n after:  %s", configBefore, configAfter)
	}
	current, err := svc.GetRawSettings()
	if err != nil {
		t.Fatalf("read live settings: %v", err)
	}
	if current.Revision.Fingerprint != live.Revision.Fingerprint {
		t.Errorf("rejected legacy save changed settings.js")
	}
}

// TestConfigHandler_SaveConfig_ConcurrentEnvEditsSurviveStaleSnapshot
// covers the "concurrent environment edits are not overwritten by a stale
// whole-config snapshot" criterion: the environment variables live in the
// same JSON store as the structured configuration, so a stale snapshot
// must not wipe an edit another session made in the meantime.
func TestConfigHandler_SaveConfig_ConcurrentEnvEditsSurviveStaleSnapshot(t *testing.T) {
	f := newRevisionSaveFixture(t)
	seed := f.svc.GetDefault()
	seed.EnvVars = []model.EnvVar{{Key: "NRCC_TEST_VAR", Value: "original"}}
	f.seedStore(t, seed)

	// Another session edits the environment through the store while the
	// operator holds a stale whole-config snapshot.
	if _, err := f.svc.Update(func(current *model.NodeRedConfig) error {
		current.EnvVars = []model.EnvVar{{Key: "NRCC_TEST_VAR", Value: "concurrent"}}
		return nil
	}); err != nil {
		t.Fatalf("concurrent env edit: %v", err)
	}

	// The operator saves the stale snapshot (no env vars in the payload).
	if w := f.save(t, configPayload(nil)); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	stored, err := f.svc.Get()
	if err != nil {
		t.Fatalf("read stored config: %v", err)
	}
	if len(stored.EnvVars) != 1 || stored.EnvVars[0].Value != "concurrent" {
		t.Errorf("concurrent environment edit was overwritten: %+v", stored.EnvVars)
	}
}
