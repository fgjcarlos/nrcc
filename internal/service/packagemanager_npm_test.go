package service

import (
	"errors"
	"testing"
)

// TestResolveNpmBinPrefersNpmBinEnv verifies that NPM_BIN env var takes precedence
// over PATH lookups and candidate paths.
func TestResolveNpmBinPrefersNpmBinEnv(t *testing.T) {
	t.Setenv("NPM_BIN", "/custom/path/npm")

	got := resolveNpmBin()
	want := "/custom/path/npm"
	if got != want {
		t.Errorf("resolveNpmBin() = %q; want %q", got, want)
	}
}

// TestNpmInstallRejectsInvalidPackage verifies that Install returns an error
// wrapping ErrInvalidPackageName before ever invoking npm.
func TestNpmInstallRejectsInvalidPackage(t *testing.T) {
	pm := NewNpmPackageManager(t.TempDir())

	err := pm.Install("evil; rm -rf /")
	if err == nil {
		t.Fatal("Install: expected error for invalid package name, got nil")
	}
	if !errors.Is(err, ErrInvalidPackageName) {
		t.Errorf("Install: expected ErrInvalidPackageName, got: %v", err)
	}
}

// TestLibraryCheckPropagatesNpmError verifies that Check returns a non-nil error
// when the npm binary cannot run, instead of silently swallowing it.
// This covers the swallowed-error bug from the previous Check implementation.
func TestLibraryCheckPropagatesNpmError(t *testing.T) {
	pm := &NpmPackageManager{
		WorkDir: t.TempDir(),
		Bin:     "/nonexistent/definitely-not-npm",
	}

	svc := NewLibraryServiceWithPackageManager(t.TempDir(), pm)

	ok, err := svc.Check("express")
	if ok {
		t.Error("Check: expected false when npm binary is missing, got true")
	}
	if err == nil {
		t.Error("Check: expected non-nil error when npm binary is missing, got nil (swallowed error bug)")
	}
}
