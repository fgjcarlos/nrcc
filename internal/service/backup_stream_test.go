package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TestValidateRestoreDestination covers the path-containment primitive that
// the restic provider restore path uses to defend against writing outside
// the operator's data volume (HIGH-007). Pure function — no I/O — so the
// table below runs in microseconds.
func TestValidateRestoreDestination(t *testing.T) {
	// Use t.TempDir() as the "root" so Abs() resolves to a real directory
	// that actually exists; ValidateRestoreDestination canonicalizes both
	// sides before the Rel() check.
	root := t.TempDir()

	cases := []struct {
		name    string
		dst     string
		root    string // empty means reuse `root`
		wantErr bool
	}{
		{name: "absolute path inside root", dst: filepath.Join(root, "nested", "sub"), wantErr: false},
		{name: "absolute path equals root", dst: root, wantErr: false},
		{name: "absolute path escapes root via sibling", dst: "/etc", wantErr: true},
		{name: "absolute path escapes via .. segment", dst: filepath.Join(root, "..", "..", "etc"), wantErr: true},
		{name: "empty destination", dst: "", wantErr: true},
		{name: "empty root", dst: "/data/x", root: "", wantErr: true},
		{name: "relative destination rejected after Abs", dst: "relative/path", wantErr: true},
		{name: "root is /, any absolute allowed", dst: "/etc", root: string(os.PathSeparator), wantErr: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := c.root
			if r == "" {
				r = root
			}
			err := ValidateRestoreDestination(c.dst, r)
			if c.wantErr && err == nil {
				t.Fatalf("expected error for dst=%q root=%q, got nil", c.dst, r)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error for dst=%q root=%q: %v", c.dst, r, err)
			}
		})
	}
}

// TestRestore_SourceHasNoEagerReadFile is a structural guard: parse the
// AST of backup.go and prove that BackupService.Restore does NOT call
// os.ReadFile or io.ReadAll on the whole archive. Catches regressions if a
// future edit reintroduces the in-memory read.
func TestRestore_SourceHasNoEagerReadFile(t *testing.T) {
	src, err := os.ReadFile("backup.go")
	if err != nil {
		t.Fatalf("read backup.go: %v", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "backup.go", src, 0)
	if err != nil {
		t.Fatalf("parse backup.go: %v", err)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name == nil {
			continue
		}
		if fn.Name.Name != "Restore" {
			continue
		}
		// Verify receiver is *BackupService.
		if len(fn.Recv.List) != 1 {
			continue
		}
		star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		ident, ok := star.X.(*ast.Ident)
		if !ok || ident.Name != "BackupService" {
			continue
		}
		if fn.Body == nil {
			continue
		}
		banned := map[string]bool{"ReadFile": true, "ReadAll": true}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if !banned[sel.Sel.Name] {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			t.Fatalf("BackupService.Restore must not call %s.%s — stream instead", pkg.Name, sel.Sel.Name)
			return false
		})
	}
}

// TestService_NoEagerWholeArchiveRead extends the structural guard to the
// Download entrypoint. Both Download and Restore must avoid pulling the
// whole archive into memory.
func TestService_NoEagerWholeArchiveRead(t *testing.T) {
	src, err := os.ReadFile("backup.go")
	if err != nil {
		t.Fatalf("read backup.go: %v", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "backup.go", src, 0)
	if err != nil {
		t.Fatalf("parse backup.go: %v", err)
	}
	banned := map[string]bool{"ReadFile": true, "ReadAll": true}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name == nil {
			continue
		}
		if fn.Name.Name != "Download" {
			continue
		}
		if fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if !banned[sel.Sel.Name] {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			t.Fatalf("BackupService.Download must not call %s.%s — stream instead", pkg.Name, sel.Sel.Name)
			return false
		})
	}
}

// TestRestore_StreamsLargeArchive_BoundedMemory proves HIGH-006: restoring
// a 50 MiB archive does not grow the Go heap by 50 MiB. We allow up to
// 8 MiB of growth for the manifest struct + per-entry hashing buffers +
// the in-progress file write.
//
// We go through svc.CreateTyped so the zip contains a real manifest with
// per-entry checksums — Restore must read the manifest, verify each
// entry, and extract without pulling the whole archive into memory.
func TestRestore_StreamsLargeArchive_BoundedMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 50 MiB memory-bound test in -short mode")
	}
	dataDir := t.TempDir()
	const payloadSize = 50 * 1024 * 1024

	// Seed a 50 MiB flows.json (the per-entry cap is 200 MiB by default).
	payload := make([]byte, payloadSize)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "flows.json"), payload, 0o644); err != nil {
		t.Fatalf("seed flows.json: %v", err)
	}

	svc := NewBackupService(dataDir)
	backup, err := svc.CreateTyped(model.BackupTypeManual, "mem-stream")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}

	// Mutate dataDir so Restore actually has work to do.
	if err := os.WriteFile(filepath.Join(dataDir, "flows.json"), []byte("[overwritten]"), 0o644); err != nil {
		t.Fatalf("overwrite: %v", err)
	}

	// Warm up GC + heap measurement before the operation.
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	if err := svc.Restore(backup.ID); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	const budget = 8 * 1024 * 1024 // 8 MiB
	delta := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	if delta > budget {
		t.Fatalf("Restore grew heap by %d bytes (budget %d). Restore is loading the whole archive into memory.", delta, budget)
	}
	t.Logf("Heap delta after 50 MiB Restore: %d bytes (budget %d)", delta, budget)

	// Sanity: the file was actually extracted with the original bytes.
	got, err := os.ReadFile(filepath.Join(dataDir, "flows.json"))
	if err != nil {
		t.Fatalf("read extracted flows.json: %v", err)
	}
	if len(got) != payloadSize {
		t.Fatalf("extracted size = %d, want %d", len(got), payloadSize)
	}
}

// TestRestore_FailsFastOnChecksumMismatch proves HIGH-008: a corrupt entry
// is detected before any file touches the dataDir.
func TestRestore_FailsFastOnChecksumMismatch(t *testing.T) {
	dataDir := t.TempDir()
	svc := NewBackupService(dataDir)
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), `[{"id":"ORIGINAL"}]`)

	backup, err := svc.CreateTyped(model.BackupTypeManual, "snap")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}

	// Open the published zip, flip a byte in flows.json, and rewrite the
	// checksum in the manifest so verifyArchiveManifest rejects.
	if err := corruptBackup(backup.Path, "flows.json"); err != nil {
		t.Fatalf("corrupt: %v", err)
	}

	// Mutate dataDir so we can prove Restore did NOT write to it.
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), `[{"id":"LIVE"}]`)

	err = svc.Restore(backup.ID)
	if err == nil {
		t.Fatal("expected checksum-mismatch error, got nil")
	}
	if !errorsIs(err, ErrBackupCorrupt) {
		t.Fatalf("expected ErrBackupCorrupt, got %v", err)
	}

	// dataDir/flows.json must still be the LIVE version — no partial restore.
	got, err := os.ReadFile(filepath.Join(dataDir, "flows.json"))
	if err != nil {
		t.Fatalf("read flows.json: %v", err)
	}
	if !strings.Contains(string(got), `"LIVE"`) {
		t.Fatalf("dataDir was modified by failed restore: %s", got)
	}

	// Staging dir must be absent or empty (cleanup-on-failure contract).
	entries, _ := os.ReadDir(dataDir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "restore-staging-") {
			t.Fatalf("staging dir leaked after failure: %s", e.Name())
		}
	}
}

// TestRestore_FailsOnMissingManifestEntry: manifest advertises files that
// are absent from the zip. The restore must refuse to do any work.
func TestRestore_FailsOnMissingManifestEntry(t *testing.T) {
	dataDir := t.TempDir()
	svc := NewBackupService(dataDir)
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), `[{"id":"X"}]`)

	backup, err := svc.CreateTyped(model.BackupTypeManual, "snap")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}

	// Strip flows.json from the zip but leave the manifest claiming it.
	if err := stripEntry(backup.Path, "flows.json"); err != nil {
		t.Fatalf("strip: %v", err)
	}

	err = svc.Restore(backup.ID)
	if err == nil {
		t.Fatal("expected error for manifest/zip mismatch, got nil")
	}
	if !errorsIs(err, ErrBackupCorrupt) {
		t.Fatalf("expected ErrBackupCorrupt, got %v", err)
	}
}

// TestRestore_Regression_ExtractsFlows is the baseline happy-path test:
// build a real backup with svc.CreateTyped, mutate the dataDir, then
// Restore and assert byte equality.
func TestRestore_Regression_ExtractsFlows(t *testing.T) {
	dataDir := t.TempDir()
	svc := NewBackupService(dataDir)
	want := `[{"id":"RESTORE_ME"}]`
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), want)

	backup, err := svc.CreateTyped(model.BackupTypeManual, "snap")
	if err != nil {
		t.Fatalf("CreateTyped: %v", err)
	}
	// Mutate so we know Restore actually overwrote it.
	writeTestFile(t, filepath.Join(dataDir, "flows.json"), `[{"id":"OTHER"}]`)

	if err := svc.Restore(backup.ID); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dataDir, "flows.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != want {
		t.Fatalf("restore content: got %s want %s", got, want)
	}
}

// TestResticProvider_Restore_RejectsEscape proves HIGH-007 at the service
// layer: even with a syntactically valid snapshot id, Restore refuses to
// hand an escaping destination to restic.
func TestResticProvider_Restore_RejectsEscape(t *testing.T) {
	// We never reach the restic binary — the validation runs first.
	// Construct a minimal provider so we exercise the real method.
	p := &ResticProvider{
		Binary: "/nonexistent",
		Repo:   "/tmp/repo",
	}
	ctx := context.Background()
	root := t.TempDir()
	err := p.Restore(ctx, "abcdef1234567890", "/etc", root)
	if err == nil {
		t.Fatal("expected error for destination escaping root")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("error should mention escape, got %v", err)
	}
}

// TestDataDir_ExposesConfiguredDirectory ensures the new getter returns
// what NewBackupService was constructed with.
func TestDataDir_ExposesConfiguredDirectory(t *testing.T) {
	want := t.TempDir()
	svc := NewBackupService(want)
	if got := svc.DataDir(); got != want {
		t.Fatalf("DataDir() = %q, want %q", got, want)
	}
}

// --- helpers ---

// writeSingleEntryZip creates a zip at path with a single file entry of
// the requested size, filled with deterministic bytes.
func writeSingleEntryZip(path string, entryName string, size int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	zw := zip.NewWriter(f)
	hdr := &zip.FileHeader{
		Name:   entryName,
		Method: zip.Deflate,
	}
	hdr.SetMode(0o644)
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	// Deterministic fill so a flaky run is reproducible.
	buf := make([]byte, 1024*1024)
	for i := range buf {
		buf[i] = byte(i % 251)
	}
	remaining := size
	for remaining > 0 {
		n := remaining
		if n > len(buf) {
			n = len(buf)
		}
		if _, err := w.Write(buf[:n]); err != nil {
			return err
		}
		remaining -= n
	}
	return zw.Close()
}

// corruptBackup rewrites the bytes of `target` inside the zip and updates
// the manifest so verifyArchiveManifest rejects (manifest checksum no
// longer matches the entry content). This requires re-zipping the archive.
func corruptBackup(zipPath, target string) error {
	// Read original archive.
	src, err := os.ReadFile(zipPath)
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return err
	}

	// Build the corrupted archive in memory.
	var dst bytes.Buffer
	zw := zip.NewWriter(&dst)
	for _, f := range zr.File {
		if f.Name == "backup-metadata.json" {
			// Rewrite manifest with the old (pre-corrupt) checksum for
			// flows.json so verify fails when it recomputes the new bytes.
			rc, err := f.Open()
			if err != nil {
				return err
			}
			var meta backupMetadata
			if err := json.NewDecoder(rc).Decode(&meta); err != nil {
				_ = rc.Close()
				return err
			}
			_ = rc.Close()
			// Leave meta.Files untouched — the checksums will mismatch.
			mb, err := json.Marshal(meta)
			if err != nil {
				return err
			}
			w, err := zw.Create("backup-metadata.json")
			if err != nil {
				return err
			}
			if _, err := w.Write(mb); err != nil {
				return err
			}
			continue
		}
		// Copy + corrupt.
		rc, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return err
		}
		if f.Name == target {
			for i := range data {
				data[i] ^= 0xFF
			}
		}
		w, err := zw.Create(f.Name)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(zipPath, dst.Bytes(), 0o644)
}

// stripEntry removes `target` from the zip (re-zips without it) but
// leaves the manifest untouched so verifyArchiveManifest sees a manifest
// entry that is absent from the zip body.
func stripEntry(zipPath, target string) error {
	src, err := os.ReadFile(zipPath)
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return err
	}
	var dst bytes.Buffer
	zw := zip.NewWriter(&dst)
	for _, f := range zr.File {
		if f.Name == target {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return err
		}
		w, err := zw.Create(f.Name)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return os.WriteFile(zipPath, dst.Bytes(), 0o644)
}

// errorsIs is a tiny shim — avoids pulling in the errors import in the
// helper scope; the production code wraps ErrBackupCorrupt with %w so
// errors.Is works once the implementation lands.
func errorsIs(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
