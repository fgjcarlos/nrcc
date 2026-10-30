package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyArtifacts_AcceptsCorrectlyPackagedDist(t *testing.T) {
	dist := writeDist(t, "1.2.3")

	version, err := parseVersion("v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := verifyArtifacts(dist, version, supportedTargets()); err != nil {
		t.Fatalf("expected the artifacts to verify, got: %v", err)
	}
}

func TestVerifyArtifacts_RejectsWrongVersion(t *testing.T) {
	dist := writeDist(t, "1.2.3")

	version, err := parseVersion("v1.2.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = verifyArtifacts(dist, version, supportedTargets())
	if err == nil {
		t.Fatal("expected the verification to fail for a mismatched version")
	}
	if !strings.Contains(err.Error(), "carries version") {
		t.Fatalf("expected a version mismatch, got: %v", err)
	}
}

func TestVerifyArtifacts_RejectsUnsupportedTarget(t *testing.T) {
	dist := writeDist(t, "1.2.3")

	unsupported := filepath.Join(dist, "nrcc-darwin-arm64")
	if err := os.WriteFile(unsupported, []byte("binary"), BinaryFileMode); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	version, err := parseVersion("v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = verifyArtifacts(dist, version, supportedTargets())
	if err == nil {
		t.Fatal("expected the verification to fail for an unsupported target")
	}
	if !strings.Contains(err.Error(), "not a supported release target") {
		t.Fatalf("expected an unsupported target failure, got: %v", err)
	}
}

func TestVerifyArtifacts_RejectsTamperedChecksum(t *testing.T) {
	dist := writeDist(t, "1.2.3")

	path := filepath.Join(dist, ChecksumFileName)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fields := strings.Fields(strings.TrimSpace(strings.SplitN(string(content), "\n", 2)[0]))
	if len(fields) != 2 {
		t.Fatalf("expected two fields in %s, got %q", ChecksumFileName, string(content))
	}
	// The digest changes with every build, so the tamper flips the last
	// hex digit to a different one instead of forcing a fixed prefix.
	last := fields[0][len(fields[0])-1]
	replacement := byte('0')
	if last == '0' {
		replacement = '1'
	}
	tamperedHash := fields[0][:len(fields[0])-1] + string(replacement)
	tampered := strings.Replace(string(content), fields[0], tamperedHash, 1)
	if err := os.WriteFile(path, []byte(tampered), ChecksumFileMode); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	version, err := parseVersion("v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = verifyArtifacts(dist, version, supportedTargets())
	if err == nil {
		t.Fatal("expected the verification to fail for a tampered checksum file")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected a checksum mismatch, got: %v", err)
	}
}

func TestVerifyArtifacts_RejectsMissingSidecar(t *testing.T) {
	dist := writeDist(t, "1.2.3")

	sidecar := filepath.Join(dist, SidecarFileName("nrcc-linux-arm64"))
	if err := os.Remove(sidecar); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	version, err := parseVersion("v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = verifyArtifacts(dist, version, supportedTargets())
	if err == nil {
		t.Fatal("expected the verification to fail for a missing sidecar")
	}
	if !strings.Contains(err.Error(), "nrcc-linux-arm64.sha256") {
		t.Fatalf("expected the missing sidecar to be reported, got: %v", err)
	}
}

func TestVerifyArtifacts_RejectsMissingBinary(t *testing.T) {
	dist := writeDist(t, "1.2.3")

	if err := os.Remove(filepath.Join(dist, "nrcc-linux-arm64")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	version, err := parseVersion("v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = verifyArtifacts(dist, version, supportedTargets())
	if err == nil {
		t.Fatal("expected the verification to fail for a missing binary")
	}
	if !strings.Contains(err.Error(), "missing artifact nrcc-linux-arm64") {
		t.Fatalf("expected the missing binary to be reported, got: %v", err)
	}
}

func TestParseStampedVersion(t *testing.T) {
	tests := []struct {
		name     string
		flags    string
		expected string
		found    bool
	}{
		{
			name:     "release flags",
			flags:    "-s -w -X main.Version=1.2.3 -extldflags=-static",
			expected: "1.2.3",
			found:    true,
		},
		{
			name:     "quoted value",
			flags:    `-ldflags="-s -w -X main.Version=1.2.3-rc.1"`,
			expected: "1.2.3-rc.1",
			found:    true,
		},
		{
			name:     "quoted value at the end of the flags",
			flags:    `-ldflags="-X main.Version=1.2.3"`,
			expected: "1.2.3",
			found:    true,
		},
		{
			name:  "no version stamp",
			flags: "-s -w",
			found: false,
		},
		{
			name:  "different variable",
			flags: "-X main.Commit=abc",
			found: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, ok := parseStampedVersion(test.flags)
			if ok != test.found {
				t.Fatalf("expected found %v, got %v", test.found, ok)
			}
			if ok && value != test.expected {
				t.Fatalf("expected %q, got %q", test.expected, value)
			}
		})
	}
}
