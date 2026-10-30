package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSupportedTargets_AreLinuxOnly(t *testing.T) {
	targets := supportedTargets()
	if len(targets) != 2 {
		t.Fatalf("expected two supported targets, got %d", len(targets))
	}

	for _, target := range targets {
		if target.OS != "linux" {
			t.Fatalf("target %s is not a Linux target", target.Platform)
		}
		if !isSupportedTarget(target.Platform) {
			t.Fatalf("target %s is not recognised as supported", target.Platform)
		}
	}

	for _, unsupported := range []string{"darwin/amd64", "darwin/arm64", "windows/amd64", "linux/386", "linux/arm/v7"} {
		if isSupportedTarget(unsupported) {
			t.Fatalf("platform %s must not be supported", unsupported)
		}
	}
}

func TestNewPlan_PrereleaseUsesCandidateChannel(t *testing.T) {
	version, err := parseVersion("v2.0.0-rc.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plan := newPlan(version, []string{"ghcr.io/fgjcarlos/nrcc"}, "abc123", "2026-01-01T00:00:00Z")

	if plan.Channel != ChannelPrerelease {
		t.Fatalf("expected channel %q, got %q", ChannelPrerelease, plan.Channel)
	}
	if !plan.Prerelease {
		t.Fatal("expected the plan to be marked as a prerelease")
	}
	if plan.Labels["org.opencontainers.image.version"] != "2.0.0-rc.1" {
		t.Fatalf("unexpected version label %q", plan.Labels["org.opencontainers.image.version"])
	}
	if plan.Labels["org.opencontainers.image.revision"] != "abc123" {
		t.Fatalf("unexpected revision label %q", plan.Labels["org.opencontainers.image.revision"])
	}
	if plan.Labels["org.opencontainers.image.created"] != "2026-01-01T00:00:00Z" {
		t.Fatalf("unexpected created label %q", plan.Labels["org.opencontainers.image.created"])
	}

	for _, tag := range plan.Tags {
		if tag == TagLatest {
			t.Fatal("a prerelease plan must not publish the latest tag")
		}
	}

	if len(plan.MetadataTags) != len(plan.Tags) {
		t.Fatalf("expected one metadata tag per tag, got %v", plan.MetadataTags)
	}
}

func TestNewPlan_StablePublishesLatest(t *testing.T) {
	version, err := parseVersion("v1.4.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plan := newPlan(version, []string{"ghcr.io/fgjcarlos/nrcc"}, "", "")

	if plan.Channel != ChannelStable {
		t.Fatalf("expected channel %q, got %q", ChannelStable, plan.Channel)
	}
	if plan.Prerelease {
		t.Fatal("a stable version must not be marked as a prerelease")
	}
	if _, ok := plan.Labels["org.opencontainers.image.revision"]; ok {
		t.Fatal("an empty revision must not produce a label")
	}

	found := false
	for _, tag := range plan.Tags {
		if tag == TagLatest {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the latest tag in %v", plan.Tags)
	}
}

func TestWriteGitHubOutput_EmitsHeredocEntries(t *testing.T) {
	version, err := parseVersion("v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	plan := newPlan(version, []string{"ghcr.io/fgjcarlos/nrcc", "docker.io/fgjcarlos/nrcc"}, "", "")

	file, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer file.Close()

	if err := writeGitHubOutput(file, plan); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := string(content)

	for _, expected := range []string{
		"version=1.2.3\n",
		"channel=stable\n",
		"prerelease=false\n",
		"images<<EOF\n",
		"tags<<EOF\n1.2.3\n1.2\nlatest\nEOF\n",
		"metadata_tags<<EOF\n",
		"labels<<EOF\n",
		"org.opencontainers.image.version=1.2.3\n",
		"platforms=linux/amd64,linux/arm64\n",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected the output to contain %q, got:\n%s", expected, output)
		}
	}

	if !strings.Contains(output, "docker.io/fgjcarlos/nrcc") {
		t.Fatalf("expected both image targets in the output, got:\n%s", output)
	}
}

func TestRunPlan_JSONOutput(t *testing.T) {
	stdout := captureStdout(t, func() {
		if err := runPlan([]string{"--version", "v1.2.3", "--image", "ghcr.io/fgjcarlos/nrcc"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, expected := range []string{`"channel": "stable"`, `"latest"`, `"linux/arm64"`} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected the output to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestRunPlan_RejectsUnsupportedVersion(t *testing.T) {
	if err := runPlan([]string{"--version", "v1.2"}); err == nil {
		t.Fatal("expected an error for a malformed version")
	}
}

// captureStdout runs fn with stdout redirected into a buffer.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	original := os.Stdout
	os.Stdout = writer

	done := make(chan string, 1)
	go func() {
		var buffer bytes.Buffer
		_, _ = buffer.ReadFrom(reader)
		done <- buffer.String()
	}()

	fn()

	os.Stdout = original
	_ = writer.Close()
	output := <-done
	_ = reader.Close()

	return output
}
