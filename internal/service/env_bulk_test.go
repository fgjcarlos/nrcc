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

// keep model symbol referenced
var _ = model.EnvVar{}

// TestImportFromNodeRed pulls only the keys NRCC does not already manage,
// translating the Node-RED types back to the NRCC vocabulary.
func TestImportFromNodeRed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "flows.json"),
		[]byte(`[{"id":"manual-global","type":"global-config","env":[{"name":"ALPHA","value":"1","type":"str"},{"name":"BETA","value":"true","type":"bool"}]}]`),
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
	if len(result.Lines) != 1 || result.Lines[0].Key != "BETA" || result.Lines[0].Type != "boolean" {
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

	if _, err := svc.ImportFromNodeRed(true, nil); err != nil {
		t.Fatalf("commit: %v", err)
	}
	config, err := svc.configSvc.Get()
	if err != nil {
		t.Fatal(err)
	}
	var beta *model.EnvVar
	for i := range config.EnvVars {
		if config.EnvVars[i].Key == "BETA" {
			beta = &config.EnvVars[i]
		}
	}
	if beta == nil {
		t.Fatal("BETA missing after commit")
	}
	if beta.Type != "boolean" || beta.Description != "imported from Node-RED" {
		t.Fatalf("unexpected BETA: %+v", beta)
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
