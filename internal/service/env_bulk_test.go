package service

import (
	"errors"
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
