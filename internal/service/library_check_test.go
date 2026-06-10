package service

import (
	"errors"
	"testing"
)

// TestLibraryServiceCheckRejectsInvalidName is the #282 regression: Check ran
// `npm view <pkg>` on the raw URL parameter without validation, so local-path,
// URL and shell-metacharacter specifiers reached npm's resolver. Check must
// reject them up front, like Install/Uninstall already do.
func TestLibraryServiceCheckRejectsInvalidName(t *testing.T) {
	svc := NewLibraryService(t.TempDir())

	bad := []string{
		"file:///etc/passwd",
		"../evil",
		"http://example.com/x",
		"foo;rm -rf /",
		"",
	}
	for _, pkg := range bad {
		ok, err := svc.Check(pkg)
		if err == nil {
			t.Errorf("Check(%q): expected validation error", pkg)
		}
		if !errors.Is(err, ErrInvalidPackageName) {
			t.Errorf("Check(%q): expected ErrInvalidPackageName, got %v", pkg, err)
		}
		if ok {
			t.Errorf("Check(%q): expected ok=false", pkg)
		}
	}
}
