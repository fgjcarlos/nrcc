package main

import "testing"

func TestParseVersion_AcceptsSupportedForms(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		semver     string
		major      int
		minor      int
		patch      int
		prerelease bool
	}{
		{name: "tagged stable", input: "v1.2.3", semver: "1.2.3", major: 1, minor: 2, patch: 3},
		{name: "bare stable", input: "1.2.3", semver: "1.2.3", major: 1, minor: 2, patch: 3},
		{name: "zero major", input: "v0.1.0", semver: "0.1.0", major: 0, minor: 1, patch: 0},
		{name: "prerelease", input: "v1.2.3-rc.1", semver: "1.2.3-rc.1", major: 1, minor: 2, patch: 3, prerelease: true},
		{name: "prerelease without tag prefix", input: "1.2.3-beta", semver: "1.2.3-beta", major: 1, minor: 2, patch: 3, prerelease: true},
		{name: "build metadata is dropped", input: "v1.2.3+build.7", semver: "1.2.3", major: 1, minor: 2, patch: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			version, err := parseVersion(test.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if version.Semver != test.semver {
				t.Fatalf("expected semver %q, got %q", test.semver, version.Semver)
			}
			if version.Major != test.major || version.Minor != test.minor || version.Patch != test.patch {
				t.Fatalf("expected version %d.%d.%d, got %d.%d.%d",
					test.major, test.minor, test.patch, version.Major, version.Minor, version.Patch)
			}
			if version.IsPrerelease() != test.prerelease {
				t.Fatalf("expected prerelease %v, got %v", test.prerelease, version.IsPrerelease())
			}
		})
	}
}

func TestParseVersion_RejectsMalformedVersions(t *testing.T) {
	inputs := []string{
		"",
		"v",
		"v1.2",
		"1.2.3.4",
		"v1.2.x",
		"v01.2.3",
		"v1.2.3-",
		"v1.2.3-rc/1",
		"latest",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			if _, err := parseVersion(input); err == nil {
				t.Fatalf("expected an error for %q", input)
			}
		})
	}
}

func TestVersionTags_StablePublishesLatest(t *testing.T) {
	version, err := parseVersion("v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tags := version.Tags()
	expected := []string{"1.2.3", "1.2", "latest"}

	if len(tags) != len(expected) {
		t.Fatalf("expected tags %v, got %v", expected, tags)
	}
	for i, tag := range expected {
		if tags[i] != tag {
			t.Fatalf("expected tags %v, got %v", expected, tags)
		}
	}
}

func TestVersionTags_PrereleaseStaysOutOfLatest(t *testing.T) {
	version, err := parseVersion("v1.2.3-rc.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tags := version.Tags()
	expected := []string{"1.2.3-rc.1", "candidate"}

	if len(tags) != len(expected) {
		t.Fatalf("expected tags %v, got %v", expected, tags)
	}
	for _, tag := range tags {
		if tag == TagLatest {
			t.Fatal("a prerelease must never publish the latest tag")
		}
		if tag == "1.2" {
			t.Fatal("a prerelease must never publish the floating major.minor tag")
		}
	}
	for i, tag := range expected {
		if tags[i] != tag {
			t.Fatalf("expected tags %v, got %v", expected, tags)
		}
	}
}
