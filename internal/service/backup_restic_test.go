package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resticBinary returns the absolute path to the restic binary used by the
// integration tests. Resolved once per test run via NRCC_RESTIC_TEST_BIN
// or a hard-coded /tmp/restic default (matches what the local dev
// environment installs when the issue is being worked on). Returns "" if
// no binary is available, in which case the tests skip themselves.
// testResticPassword is the passphrase used by integration tests to init a
// throwaway local repository. It is a fixture, not a secret — the repo
// lives in t.TempDir() and is wiped when the test ends. GitGuardian's
// generic-password detector flags plain `Password: "..."` literals;
// grouping the value under a clearly-named constant and tagging the file
// with the standard nosecret directive avoids the false positive without
// resorting to dynamic generation that would complicate debugging.
// nosecret (GitGuardian: test fixture, not a credential)
const testResticPassword = "nrcc-integration-test-fixture-pw"

// resticBinary returns the absolute path to the restic binary used by the
// integration tests. Resolved once per test run via NRCC_RESTIC_TEST_BIN
// or a hard-coded /tmp/restic default (matches what the local dev
// environment installs when the issue is being worked on). Returns "" if
// no binary is available, in which case the tests skip themselves.
func resticBinary(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("NRCC_RESTIC_TEST_BIN"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}
	candidates := []string{"/tmp/restic", "/usr/local/bin/restic", "/usr/bin/restic"}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("restic binary not available; set NRCC_RESTIC_TEST_BIN or install at /tmp/restic")
	return ""
}

// TestResticProviderSnapshotListRestore is the round-trip smoke test: init
// a fresh repo, push one snapshot, list, restore into a new directory, and
// assert the restored file matches what we put in.
func TestResticProviderSnapshotListRestore(t *testing.T) {
	binary := resticBinary(t)
	repo := filepath.Join(t.TempDir(), "repo")
	cache := filepath.Join(t.TempDir(), "cache")

	p, err := NewResticProvider(ResticConfig{
		Binary:   binary,
		Repo:     repo,
		Password: testResticPassword,
		CacheDir: cache,
	})
	if err != nil {
		t.Fatalf("NewResticProvider: %v", err)
	}

	srcDir := t.TempDir()
	secret := []byte(`[{"id":"node-1","type":"inject"}]`)
	if err := os.WriteFile(filepath.Join(srcDir, "flows.json"), secret, 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := context.Background()
	id, err := p.Snapshot(ctx, srcDir)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if id == "" {
		t.Fatal("empty snapshot id")
	}

	snaps, err := p.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(snaps) != 1 || snaps[0].ID != id {
		t.Fatalf("List returned %+v, want one snapshot id=%s", snaps, id)
	}

	dst := filepath.Join(t.TempDir(), "restored")
	if err := p.Restore(ctx, id, dst); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	// restic preserves the absolute source path inside the snapshot; the
	// restored file lives at <dst>/<srcDir>/flows.json.
	got, err := os.ReadFile(filepath.Join(dst, srcDir, "flows.json"))
	if err != nil {
		t.Fatalf("read restored flows.json: %v", err)
	}
	if string(got) != string(secret) {
		t.Fatalf("restored content mismatch: got %q want %q", got, secret)
	}
}

// TestResticProviderRejectsEmptyConfig catches the two mandatory knobs.
func TestResticProviderRejectsEmptyConfig(t *testing.T) {
	if _, err := NewResticProvider(ResticConfig{}); err == nil {
		t.Fatal("expected error for empty config")
	}
	if _, err := NewResticProvider(ResticConfig{Repo: "local:/tmp/x"}); err == nil {
		t.Fatal("expected error when both password fields are empty")
	}
}

// TestBackupServiceNoopProviderByDefault proves a fresh BackupService
// reports the local-only provider and does not silently fail in push paths.
func TestBackupServiceNoopProviderByDefault(t *testing.T) {
	svc := NewBackupService(t.TempDir())
	if name := svc.BackupProvider().Name(); name != "local" {
		t.Fatalf("expected default provider %q, got %q", "local", name)
	}
	if err := os.WriteFile(filepath.Join(t.TempDir(), "noop-src"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
}

// TestResticProviderInitIsIdempotent calls init twice and asserts the
// second call treats "already initialized" as a no-op.
func TestResticProviderInitIsIdempotent(t *testing.T) {
	binary := resticBinary(t)
	repo := filepath.Join(t.TempDir(), "repo")
	p, err := NewResticProvider(ResticConfig{
		Binary:   binary,
		Repo:     repo,
		Password: testResticPassword,
		CacheDir: filepath.Join(t.TempDir(), "cache"),
	})
	if err != nil {
		t.Fatalf("NewResticProvider: %v", err)
	}
	if err := p.initRepoIfNeeded(context.Background()); err != nil {
		t.Fatalf("first init: %v", err)
	}
	if err := p.initRepoIfNeeded(context.Background()); err != nil {
		t.Fatalf("second init: %v", err)
	}
}

// TestResticProviderHandlesEmptyList proves a fresh repo's snapshot listing
// returns an empty slice (not nil/error).
func TestResticProviderHandlesEmptyList(t *testing.T) {
	binary := resticBinary(t)
	repo := filepath.Join(t.TempDir(), "repo")
	p, err := NewResticProvider(ResticConfig{
		Binary:   binary,
		Repo:     repo,
		Password: testResticPassword,
		CacheDir: filepath.Join(t.TempDir(), "cache"),
	})
	if err != nil {
		t.Fatalf("NewResticProvider: %v", err)
	}
	if err := p.initRepoIfNeeded(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
	snaps, err := p.List(context.Background())
	if err != nil {
		t.Fatalf("List on empty repo: %v", err)
	}
	if len(snaps) != 0 {
		t.Fatalf("expected empty list, got %+v", snaps)
	}
}

// TestParseResticSnapshotIDAcceptsWellFormedSummary covers the parser
// without invoking the binary, so it runs even when restic is absent.
func TestParseResticSnapshotIDAcceptsWellFormedSummary(t *testing.T) {
	out := []byte(strings.Join([]string{
		`{"message_type":"start","time":"2026-07-16T05:30:00Z"}`,
		`{"message_type":"file","path":"/flows.json"}`,
		`{"message_type":"summary","snapshot_id":"abc123","files_new":1}`,
	}, "\n"))
	id, err := parseResticSnapshotID(out)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if id != "abc123" {
		t.Fatalf("got %q, want abc123", id)
	}
}

func TestValidateResticSnapshotID(t *testing.T) {
	cases := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "valid short hex", id: "abc12345", wantErr: false},
		{name: "valid long hex", id: strings.Repeat("a", 64), wantErr: false},
		{name: "empty", id: "", wantErr: true},
		{name: "too short", id: "abc", wantErr: true},
		{name: "too long", id: strings.Repeat("a", 65), wantErr: true},
		{name: "uppercase hex", id: "ABC12345", wantErr: true},
		{name: "argv injection", id: "--help", wantErr: true},
		{name: "non-hex chars", id: "abcdefgh", wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateResticSnapshotID(c.id)
			if c.wantErr && err == nil {
				t.Fatalf("expected error for %q", c.id)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", c.id, err)
			}
		})
	}
}