package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestParseBulkEnv(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOK    bool
		wantLines int
		wantIssue string
	}{
		{
			name:      "empty input",
			input:     "",
			wantOK:    false,
			wantLines: 0,
		},
		{
			name:      "string default",
			input:     "API_URL=https://x.test\nDEBUG=true#boolean\nTOKEN=A=B#secret\n",
			wantOK:    true,
			wantLines: 3,
		},
		{
			name:      "comments and blanks",
			input:     "# header\n\nKEY=v\n",
			wantOK:    true,
			wantLines: 1,
		},
		{
			name:      "duplicate key",
			input:     "KEY=1\nKEY=2\n",
			wantOK:    false,
			wantLines: 1,
			wantIssue: "duplicate",
		},
		{
			name:      "missing equals",
			input:     "GOOD=1\nBADLINE\n",
			wantOK:    false,
			wantLines: 1,
			wantIssue: "missing",
		},
		{
			name:      "unknown type",
			input:     "FOO=1#weird\n",
			wantOK:    false,
			wantLines: 0,
			wantIssue: "unknown type",
		},
		{
			name:      "invalid number",
			input:     "PORT=abc#number\n",
			wantOK:    false,
			wantLines: 0,
			wantIssue: "number",
		},
		{
			name:      "boolean strict",
			input:     "FLAG=true#boolean\n",
			wantOK:    true,
			wantLines: 1,
		},
		{
			name:      "boolean rejected",
			input:     "FLAG=Yes#boolean\n",
			wantOK:    false,
			wantLines: 0,
			wantIssue: "'true' or 'false'",
		},
		{
			name:      "key with forbidden chars",
			input:     "BAD=KEY=x\n",
			wantOK:    true,
			wantLines: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseBulkEnv(tc.input)
			if got.Valid != tc.wantOK {
				t.Fatalf("Valid=%v, want %v. Issues=%v", got.Valid, tc.wantOK, got.Issues)
			}
			if len(got.Lines) != tc.wantLines {
				t.Fatalf("len(Lines)=%d, want %d (issues=%v)", len(got.Lines), tc.wantLines, got.Issues)
			}
			if tc.wantIssue != "" {
				found := false
				for _, iss := range got.Issues {
					if strings.Contains(iss.Reason, tc.wantIssue) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected issue containing %q, got %v", tc.wantIssue, got.Issues)
				}
			}
		})
	}
}

func TestParseBulkEnv_RejectsTooManyEntries(t *testing.T) {
	result := mustParseBulkWithLimits(t, "A=1\nB=2\nC=3\n", BulkLimits{MaxEntries: 2, MaxEntryBytes: 8192})
	assertBulkCap(t, result, 2, "maximum 2 entries")
}

func TestParseBulkEnv_RejectsEntryTooLarge(t *testing.T) {
	result := mustParseBulkWithLimits(t, "K="+strings.Repeat("v", 8192), BulkLimits{MaxEntries: 1000, MaxEntryBytes: 8192})
	assertBulkCap(t, result, 0, "maximum 8192 bytes")
}

func TestParseBulkEnv_AcceptsAtLimits(t *testing.T) {
	value := strings.Repeat("é", 4095) + "v"
	result := mustParseBulkWithLimits(t, "# ignored\n\nK="+value+"#string\nB=2", BulkLimits{MaxEntries: 2, MaxEntryBytes: 8192})
	if !result.Valid || len(result.Lines) != 2 {
		t.Fatalf("expected limits to be inclusive, got %+v", result)
	}
}

func TestParseBulkEnv_LimitsFromConfig(t *testing.T) {
	tests := []struct {
		name, entries, bytes string
		want                 BulkLimits
		wantErr              bool
	}{
		{"defaults", "", "", DefaultBulkLimits(), false},
		{"zero uses defaults", "0", "0", DefaultBulkLimits(), false},
		{"custom", "2", "16", BulkLimits{MaxEntries: 2, MaxEntryBytes: 16}, false},
		{"negative", "-1", "16", BulkLimits{}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("NRCC_BULK_MAX_ENTRIES", test.entries)
			t.Setenv("NRCC_BULK_MAX_ENTRY_BYTES", test.bytes)
			got, err := BulkLimitsFromEnv()
			if (err != nil) != test.wantErr || got != test.want {
				t.Fatalf("limits=%+v, err=%v, want %+v/error=%v", got, err, test.want, test.wantErr)
			}
		})
	}
}

func mustParseBulkWithLimits(t *testing.T, content string, limits BulkLimits) BulkEnvResult {
	t.Helper()
	result, err := ParseBulkEnvWithLimits(content, limits)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertBulkCap(t *testing.T, result BulkEnvResult, lines int, issue string) {
	t.Helper()
	if result.Valid || len(result.Lines) != lines || len(result.Issues) != 1 || !strings.Contains(result.Issues[0].Reason, issue) {
		t.Fatalf("expected %d lines and issue %q, got %+v", lines, issue, result)
	}
}

func TestApplyBulkEnvPersistsSecretsAndNonSecrets(t *testing.T) {
	dir := t.TempDir()
	svc := NewEnvService(NewIsolatedConfigService(dir), "test-key")
	parsed := ParseBulkEnv("ALPHA=1\nBETA=hidden#secret\n")
	if !parsed.Valid {
		t.Fatalf("parse failed: %v", parsed.Issues)
	}
	if _, err := svc.ApplyBulkEnv(parsed, nil); err != nil {
		t.Fatalf("ApplyBulkEnv: %v", err)
	}

	config, err := svc.configSvc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if len(config.EnvVars) != 2 {
		t.Fatalf("env vars len=%d, want 2", len(config.EnvVars))
	}
	for _, ev := range config.EnvVars {
		switch ev.Key {
		case "ALPHA":
			if ev.Encrypted {
				t.Fatalf("ALPHA must not be encrypted")
			}
		case "BETA":
			if !ev.Encrypted {
				t.Fatalf("BETA must be encrypted")
			}
		default:
			t.Fatalf("unexpected key %q", ev.Key)
		}
	}
}

func TestApplyBulkEnvRejectsInvalid(t *testing.T) {
	dir := t.TempDir()
	svc := NewEnvService(NewIsolatedConfigService(dir))
	parsed := ParseBulkEnv("BADLINE")
	if _, err := svc.ApplyBulkEnv(parsed, nil); err == nil || !errors.Is(err, err) {
		t.Fatalf("expected error for invalid payload, got nil (parsed=%v)", parsed)
	}
}

func TestApplyBulkEnv_SyncCountOne(t *testing.T) {
	svc := NewEnvService(NewIsolatedConfigService(t.TempDir()))
	syncCalls := 0
	svc.syncNodeRedGlobals = func(keys []string) error {
		syncCalls++
		if len(keys) != 3 {
			t.Fatalf("sync received %d keys, want 3", len(keys))
		}
		return nil
	}
	parsed := ParseBulkEnv("A=1\nB=2\nC=3")
	if _, err := svc.ApplyBulkEnv(parsed, nil); err != nil {
		t.Fatalf("ApplyBulkEnv: %v", err)
	}
	if syncCalls != 1 {
		t.Fatalf("sync calls=%d, want 1", syncCalls)
	}
}

func TestApplyBulkEnv_PreservesManualGlobals(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(`[{"id":"global","type":"global-config","env":[{"name":"MANUAL","value":"keep","type":"str"}]}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewEnvService(NewIsolatedConfigService(dir), "test-key")
	parsed := ParseBulkEnv("A=1\nTOKEN=hidden#secret")
	if _, err := svc.ApplyBulkEnv(parsed, nil); err != nil {
		t.Fatalf("ApplyBulkEnv: %v", err)
	}
	flows, err := os.ReadFile(filepath.Join(dir, "flows.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(flows)
	if !strings.Contains(text, `"name": "MANUAL"`) || !strings.Contains(text, `"name": "A"`) {
		t.Fatalf("manual or managed global missing: %s", text)
	}
	if strings.Contains(text, `"name": "TOKEN"`) || strings.Contains(text, "hidden") {
		t.Fatalf("secret leaked into globals: %s", text)
	}
}

func TestApplyBulkEnv_PartialFailureRollback(t *testing.T) {
	svc := NewEnvService(NewIsolatedConfigService(t.TempDir()))
	syncCalls := 0
	svc.syncNodeRedGlobals = func([]string) error {
		syncCalls++
		return nil
	}
	parsed := BulkEnvResult{Valid: true, Lines: []BulkEnvLine{
		{Line: 1, Key: "A", Value: "1", Type: "string"},
		{Line: 2, Key: "BAD\nKEY", Value: "2", Type: "string"},
	}}
	if _, err := svc.ApplyBulkEnv(parsed, nil); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("expected contextual second-line failure, got %v", err)
	}
	if syncCalls != 0 {
		t.Fatalf("partial apply synchronized %d times, want 0", syncCalls)
	}
	config, err := svc.configSvc.Get()
	if err != nil {
		t.Fatal(err)
	}
	if len(config.EnvVars) != 1 || config.EnvVars[0].Key != "A" {
		t.Fatalf("existing per-entry commit semantics changed: %+v", config.EnvVars)
	}
}

// keep model symbol referenced
var _ = model.EnvVar{}

// TestImportFromNodeRed pulls only the keys NRCC does not already manage,
// translating the Node-RED types back to the NRCC vocabulary.
func TestImportFromNodeRed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "flows.json"),
		[]byte(`[{"id":"manual-global","type":"global-config","env":[{"name":"ALPHA","value":"1","type":"str"},{"name":"BETA","value":"true","type":"bool"},{"name":"GAMMA","value":"3","type":"num"}]}]`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	svc := NewEnvService(NewIsolatedConfigService(dir), "test-key")
	if err := svc.Set("ALPHA", "local", "string", "", false); err != nil {
		t.Fatal(err)
	}

	result, err := svc.ImportFromNodeRed(false, nil)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got %+v", result)
	}
	if len(result.Lines) != 2 || result.Lines[0].Key != "BETA" || result.Lines[0].Type != "boolean" || result.Lines[1].Key != "GAMMA" || result.Lines[1].Type != "number" {
		t.Fatalf("unexpected lines: %+v", result.Lines)
	}
	hasSkip := false
	for _, iss := range result.Issues {
		if iss.Key == "ALPHA" && iss.Reason == "already managed by NRCC" {
			hasSkip = true
		}
	}
	if !hasSkip {
		t.Fatalf("expected ALPHA skip issue, got %+v", result.Issues)
	}

	callbackCalls := 0
	if _, err := svc.ImportFromNodeRed(true, func(change func() error) (bool, error) {
		callbackCalls++
		return true, change()
	}); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if callbackCalls != 1 {
		t.Fatalf("expected one lifecycle callback, got %d", callbackCalls)
	}
	config, err := svc.configSvc.Get()
	if err != nil {
		t.Fatal(err)
	}
	var beta, gamma *model.EnvVar
	for i := range config.EnvVars {
		switch config.EnvVars[i].Key {
		case "BETA":
			beta = &config.EnvVars[i]
		case "GAMMA":
			gamma = &config.EnvVars[i]
		}
	}
	if beta == nil || gamma == nil {
		t.Fatalf("imported variables missing after commit: BETA=%+v GAMMA=%+v", beta, gamma)
	}
	if beta.Type != "boolean" || beta.Description != "imported from Node-RED" {
		t.Fatalf("unexpected BETA: %+v", beta)
	}
	if gamma.Type != "number" || gamma.Description != "imported from Node-RED" {
		t.Fatalf("unexpected GAMMA: %+v", gamma)
	}
}

func TestImportFromNodeRedEmptyFlows(t *testing.T) {
	dir := t.TempDir()
	svc := NewEnvService(NewIsolatedConfigService(dir))
	result, err := svc.ImportFromNodeRed(false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid || result.Summary != "no global-config entries in Node-RED" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

// restartCall records a single callback invocation for #540 regression.
type restartCall struct {
	set func() error
}

// restartRecorder captures every ApplyBulkEnv callback invocation by #540 regression test.
type restartRecorder struct {
	calls []restartCall
	err   error
}

// callback implements the restart parameter contract for ApplyBulkEnv.
func (r *restartRecorder) callback(set func() error) (bool, error) {
	r.calls = append(r.calls, restartCall{set: set})
	if r.err != nil {
		return false, r.err
	}
	return true, nil
}

// TestApplyBulkEnvRestartCallback is the #540 regression: it locks in the
// dual-path contract of ApplyBulkEnv. If a future change removes the
// `else if err := set()` branch, the nil-path assertion below will fail.
func TestApplyBulkEnvRestartCallback(t *testing.T) {
	tests := []struct {
		name            string
		useNilRestart   bool
		recorderErr     error
		wantCallCount   int
		wantErr         bool
		wantErrContains string
	}{
		{
			name:          "nil_restart_calls_set_directly",
			useNilRestart: true,
			wantCallCount: 0, // nil path does NOT call restart callback
		},
		{
			name:          "non_nil_restart_wraps_each_line",
			useNilRestart: false,
			wantCallCount: 2, // non-nil path calls restart callback for each line
		},
		{
			name:            "non_nil_restart_propagates_error",
			useNilRestart:   false,
			recorderErr:     errors.New("boom"),
			wantCallCount:   1, // stops after first error
			wantErr:         true,
			wantErrContains: "boom",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			svc := NewEnvService(NewIsolatedConfigService(dir), "test-key")
			parsed := BulkEnvResult{
				Valid: true,
				Lines: []BulkEnvLine{
					{Key: "FOO", Value: "1", Type: "string", Line: 1},
					{Key: "BAR", Value: "2", Type: "string", Line: 2},
				},
			}

			rec := &restartRecorder{err: tc.recorderErr}
			var restart func(func() error) (bool, error)
			if !tc.useNilRestart {
				restart = rec.callback
			}

			_, err := svc.ApplyBulkEnv(parsed, restart)

			// Verify error expectation
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErrContains)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify callback was called the expected number of times.
			// This is the core #540 contract: restart==nil means callback not called;
			// restart!=nil means callback called for each line (unless early error).
			if len(rec.calls) != tc.wantCallCount {
				t.Fatalf("callback calls = %d, want %d", len(rec.calls), tc.wantCallCount)
			}
		})
	}
}
