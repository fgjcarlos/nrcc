//go:build linux

package service

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIsInteractiveTerminal_StdinIsDevNull exercises the docker-run hang:
// when stdin is redirected to /dev/null, isInteractiveTerminal must return
// false even though /dev/null is a character device. We can't safely swap
// the test runner's stdin fd, so we cover the predicate that catches it.
func TestIsDevNull_StatDevNull(t *testing.T) {
	info, err := os.Stat(os.DevNull)
	if err != nil {
		t.Skipf("cannot stat /dev/null on this host: %v", err)
	}
	if !isDevNull(info) {
		t.Fatal("isDevNull(/dev/null) = false, want true")
	}
}

func TestIsDevNull_StatRegularFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "plain.txt")
	if err := os.WriteFile(path, []byte("hi"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if isDevNull(info) {
		t.Fatal("isDevNull(regular file) = true, want false")
	}
}