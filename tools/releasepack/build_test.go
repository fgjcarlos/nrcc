package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildSampleBinary compiles a tiny Go program that carries the same
// version variable as the nrcc binary, so the verification runs against a
// real ELF binary with real build information.
func buildSampleBinary(t *testing.T, version string) string {
	t.Helper()

	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	content := "package main\n\nimport \"fmt\"\n\nvar Version string = \"dev\"\n\nfunc main() {\n\tfmt.Println(Version)\n}\n"
	if err := os.WriteFile(source, []byte(content), 0o600); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	init := exec.Command("go", "mod", "init", "example.com/sample")
	init.Dir = dir
	if output, err := init.CombinedOutput(); err != nil {
		t.Skipf("go mod init is unavailable: %v (%s)", err, string(output))
	}

	output := filepath.Join(dir, "sample")
	build := exec.Command("go", "build",
		"-buildvcs=false",
		"-ldflags", "-s -w -X main.Version="+version+" -extldflags=-static",
		"-o", output, ".")
	build.Dir = dir
	if combined, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v (%s)", err, string(combined))
	}

	return output
}

// writeDist builds a dist directory that contains the expected artifacts.
func writeDist(t *testing.T, version string) string {
	t.Helper()

	dist := t.TempDir()
	binary := buildSampleBinary(t, version)

	for _, target := range supportedTargets() {
		content, err := os.ReadFile(binary)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		path := filepath.Join(dist, target.Binary)
		if err := os.WriteFile(path, content, BinaryFileMode); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if err := writeChecksums(dist, supportedTargets()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return dist
}

func TestWriteChecksums_WritesNormalizedSidecars(t *testing.T) {
	dist := writeDist(t, "1.2.3")

	sidecar, err := os.ReadFile(filepath.Join(dist, SidecarFileName("nrcc-linux-amd64")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fields := strings.Fields(strings.TrimSpace(string(sidecar)))
	if len(fields) != 2 {
		t.Fatalf("expected two fields in the sidecar, got %q", string(sidecar))
	}
	if fields[1] != "nrcc" {
		t.Fatalf("expected the normalised name nrcc, got %q", fields[1])
	}

	sum, err := fileChecksum(filepath.Join(dist, "nrcc-linux-amd64"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields[0] != sum {
		t.Fatalf("expected the sidecar checksum %q, got %q", sum, fields[0])
	}
}
