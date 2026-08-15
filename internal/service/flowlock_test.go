package service

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestFlockExclusive_AcquiresAndReleases(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "flows.json")
	if err := os.WriteFile(target, []byte("[]"), 0o600); err != nil {
		t.Fatalf("seed flows.json: %v", err)
	}

	f, err := flockExclusive(target)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if f == nil {
		t.Fatal("flockExclusive returned a nil *os.File on success")
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestFlockExclusive_LockNBContentionReturnsError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "flows.json")
	if err := os.WriteFile(target, []byte("[]"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	first, err := flockExclusive(target)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	t.Cleanup(func() { _ = first.Close() })

	_, err = flockExclusive(target)
	if err == nil {
		t.Fatal("expected contention error from second flockExclusive")
	}
	if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
		t.Fatalf("expected EWOULDBLOCK or EAGAIN, got %v", err)
	}
}

func TestFlockExclusive_CloseReleasesLock(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "flows.json")
	if err := os.WriteFile(target, []byte("[]"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	f1, err := flockExclusive(target)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := f1.Close(); err != nil {
		t.Fatalf("close first: %v", err)
	}

	f2, err := flockExclusive(target)
	if err != nil {
		t.Fatalf("second acquire after close: %v", err)
	}
	if err := f2.Close(); err != nil {
		t.Fatalf("close second: %v", err)
	}
}

func TestFlockExclusive_LockFileIsSiblingOfTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "flows.json")
	if err := os.WriteFile(target, []byte("[]"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	f, err := flockExclusive(target)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer func() { _ = f.Close() }()

	lockPath := target + ".lock"
	info, err := os.Stat(lockPath)
	if err != nil {
		t.Fatalf("expected sibling lockfile at %s: %v", lockPath, err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("lockfile perm = %o, want 600", info.Mode().Perm())
	}
}

func TestOpenNoFollow_OpensRegularFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "data.bin")
	payload := []byte("plain bytes for read")
	if err := os.WriteFile(target, payload, 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	f, err := openNoFollow(target)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	got, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("read bytes = %q, want %q", got, payload)
	}
}

func TestOpenNoFollow_RejectsSymlinkToExistingTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(target, []byte("sensitive"), 0o600); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	link := filepath.Join(dir, "looks-like-regular")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	f, err := openNoFollow(link)
	if err == nil {
		if f != nil {
			_ = f.Close()
		}
		t.Fatal("expected symlink rejection")
	}
}

func TestOpenNoFollow_RejectsSymlinkToMissingTarget(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "dangling")
	if err := os.Symlink(filepath.Join(dir, "does-not-exist"), link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	_, err := openNoFollow(link)
	if err == nil {
		t.Fatal("expected rejection of dangling symlink")
	}
}