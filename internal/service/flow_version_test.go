package service

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFlows(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "flows.json"), []byte(content), 0644); err != nil {
		t.Fatalf("write flows.json: %v", err)
	}
}

func TestFlowVersion_CaptureAndList(t *testing.T) {
	dir := t.TempDir()
	writeTestFlows(t, dir, `[{"id":"flow1","type":"tab","label":"Test"}]`)

	svc := NewFlowVersionService(dir)

	if err := svc.CaptureNow(); err != nil {
		t.Fatalf("CaptureNow: %v", err)
	}

	versions, err := svc.ListVersions()
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}

	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}

	if versions[0].NodeCount != 1 {
		t.Errorf("nodeCount = %d, want 1", versions[0].NodeCount)
	}
	if versions[0].Hash == "" {
		t.Error("hash should not be empty")
	}
}

func TestFlowVersion_DuplicateCaptureSkipped(t *testing.T) {
	dir := t.TempDir()
	writeTestFlows(t, dir, `[{"id":"flow1","type":"tab"}]`)

	svc := NewFlowVersionService(dir)
	svc.captureIfChanged()
	svc.captureIfChanged()

	versions, _ := svc.ListVersions()
	if len(versions) != 1 {
		t.Errorf("duplicate captures should be skipped, got %d versions", len(versions))
	}
}

func TestFlowVersion_NewCaptureOnChange(t *testing.T) {
	dir := t.TempDir()
	writeTestFlows(t, dir, `[{"id":"flow1","type":"tab"}]`)

	svc := NewFlowVersionService(dir)
	svc.captureIfChanged()

	writeTestFlows(t, dir, `[{"id":"flow1","type":"tab"},{"id":"flow2","type":"tab"}]`)
	svc.captureIfChanged()

	versions, _ := svc.ListVersions()
	if len(versions) != 2 {
		t.Errorf("expected 2 versions after change, got %d", len(versions))
	}
}

func TestFlowVersion_DiffAddedRemoved(t *testing.T) {
	dir := t.TempDir()
	svc := NewFlowVersionService(dir)

	writeTestFlows(t, dir, `[{"id":"n1","type":"inject","label":"Start"}]`)
	svc.CaptureNow()
	v1, _ := svc.ListVersions()
	firstID := v1[0].ID

	writeTestFlows(t, dir, `[{"id":"n2","type":"debug","label":"End"}]`)
	svc.CaptureNow()
	v2, _ := svc.ListVersions()

	// Find the new version ID (the one that's not firstID)
	var secondID string
	for _, v := range v2 {
		if v.ID != firstID {
			secondID = v.ID
			break
		}
	}

	diff, err := svc.DiffVersions(firstID, secondID)
	if err != nil {
		t.Fatalf("DiffVersions: %v", err)
	}

	if len(diff.Added) != 1 || diff.Added[0].ID != "n2" {
		t.Errorf("expected n2 added, got %+v", diff.Added)
	}
	if len(diff.Removed) != 1 || diff.Removed[0].ID != "n1" {
		t.Errorf("expected n1 removed, got %+v", diff.Removed)
	}
}

func TestFlowVersion_DiffModified(t *testing.T) {
	dir := t.TempDir()
	svc := NewFlowVersionService(dir)

	writeTestFlows(t, dir, `[{"id":"n1","type":"inject","name":"old"}]`)
	svc.CaptureNow()
	v1, _ := svc.ListVersions()
	firstID := v1[0].ID

	writeTestFlows(t, dir, `[{"id":"n1","type":"inject","name":"new"}]`)
	svc.CaptureNow()
	v2, _ := svc.ListVersions()

	var secondID string
	for _, v := range v2 {
		if v.ID != firstID {
			secondID = v.ID
			break
		}
	}

	diff, err := svc.DiffVersions(firstID, secondID)
	if err != nil {
		t.Fatalf("DiffVersions: %v", err)
	}

	if len(diff.Modified) != 1 {
		t.Fatalf("expected 1 modified, got %d", len(diff.Modified))
	}
	if diff.Modified[0].ID != "n1" {
		t.Errorf("modified id = %q, want n1", diff.Modified[0].ID)
	}
}

func TestFlowVersion_Revert(t *testing.T) {
	dir := t.TempDir()
	svc := NewFlowVersionService(dir)

	original := `[{"id":"n1","type":"inject"}]`
	writeTestFlows(t, dir, original)
	svc.CaptureNow()
	versions, _ := svc.ListVersions()
	originalID := versions[0].ID

	writeTestFlows(t, dir, `[{"id":"n1","type":"inject"},{"id":"n2","type":"debug"}]`)
	svc.CaptureNow()

	if err := svc.Revert(originalID); err != nil {
		t.Fatalf("Revert: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "flows.json"))
	if string(data) != original {
		t.Errorf("flows.json not reverted, got %s", data)
	}
}

func TestFlowVersion_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	svc := NewFlowVersionService(dir)

	_, err := svc.GetVersion("../../etc/passwd")
	if err == nil {
		t.Error("should reject path traversal")
	}
}

func TestFlowVersion_VersionFilePermissions(t *testing.T) {
	dir := t.TempDir()
	writeTestFlows(t, dir, `[{"id":"n1","type":"tab"}]`)

	svc := NewFlowVersionService(dir)
	svc.CaptureNow()

	entries, _ := os.ReadDir(filepath.Join(dir, versionDir))
	for _, e := range entries {
		info, _ := e.Info()
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("version file permissions = %o, want 0600", perm)
		}
	}
}
