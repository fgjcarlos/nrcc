package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveJWTSecret_UsesEnvVar(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-real-production-secret-that-is-long-enough")
	secret, err := resolveJWTSecret(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret != "my-real-production-secret-that-is-long-enough" {
		t.Fatalf("expected env var value, got %q", secret)
	}
}

func TestResolveJWTSecret_RejectsPlaceholders(t *testing.T) {
	placeholders := []string{
		"cc-secret-change-in-production",
		"change-me-in-production",
		"dev-secret-not-for-production",
	}

	for _, p := range placeholders {
		t.Run(p, func(t *testing.T) {
			t.Setenv("JWT_SECRET", p)
			_, err := resolveJWTSecret(t.TempDir())
			if err == nil {
				t.Fatalf("expected error for placeholder %q", p)
			}
		})
	}
}

func TestResolveJWTSecret_GeneratesAndPersists(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	dataDir := t.TempDir()

	secret, err := resolveJWTSecret(dataDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secret) < 64 {
		t.Fatalf("generated secret too short: %d chars", len(secret))
	}

	// Verify file exists with 0600
	info, err := os.Stat(filepath.Join(dataDir, "jwt_secret"))
	if err != nil {
		t.Fatalf("secret file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600 permissions, got %o", info.Mode().Perm())
	}

	// Second call should return same secret
	secret2, err := resolveJWTSecret(dataDir)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if secret2 != secret {
		t.Fatal("expected persisted secret to be reused")
	}
}
