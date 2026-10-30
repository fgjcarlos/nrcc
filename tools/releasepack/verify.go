package main

import (
	"bytes"
	"debug/buildinfo"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// runVerify implements the "verify" subcommand.
func runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	version := fs.String("version", "", "release version, for example v1.2.3")
	dist := fs.String("dist", "dist", "directory the artifacts live in")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *version == "" {
		return errors.New("--version is required")
	}

	parsed, err := parseVersion(*version)
	if err != nil {
		return err
	}

	if err := verifyArtifacts(*dist, parsed, supportedTargets()); err != nil {
		return err
	}

	fmt.Printf("verified %d artifacts in %s for version %s\n", len(supportedTargets()), *dist, parsed.Semver)
	return nil
}

// verifyArtifacts verifies the produced release artifacts.
//
// The checks are the ones the release contract requires before anything is
// published: the file set matches the supported targets, the checksums
// match the binaries, the sidecars match the combined checksum file and
// every binary carries the stamped release version.
func verifyArtifacts(dist string, version Version, targets []Target) error {
	var problems []error

	expected := map[string]bool{}
	for _, target := range targets {
		expected[target.Binary] = true
	}

	problems = append(problems, verifyFileSet(dist, expected)...)
	problems = append(problems, verifyChecksums(dist, expected)...)
	problems = append(problems, verifySidecars(dist, expected)...)
	problems = append(problems, verifyVersionMetadata(dist, version, targets)...)

	if len(problems) > 0 {
		return errors.Join(problems...)
	}
	return nil
}

// verifyFileSet checks that every expected binary exists, is executable and
// that no artifact of an unsupported target is present.
func verifyFileSet(dist string, expected map[string]bool) []error {
	var problems []error

	for _, name := range sortedNames(expected) {
		path := filepath.Join(dist, name)
		info, err := os.Stat(path)
		if err != nil {
			problems = append(problems, fmt.Errorf("missing artifact %s: %w", name, err))
			continue
		}
		if !info.Mode().IsRegular() {
			problems = append(problems, fmt.Errorf("artifact %s is not a regular file", name))
			continue
		}
		if info.Mode().Perm()&0o111 == 0 {
			problems = append(problems, fmt.Errorf("artifact %s is not executable (mode %s)", name, info.Mode().Perm()))
		}
	}

	entries, err := os.ReadDir(dist)
	if err != nil {
		return append(problems, fmt.Errorf("read dist directory: %w", err))
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "nrcc-") {
			continue
		}
		if expected[name] || expected[strings.TrimSuffix(name, ".sha256")] {
			continue
		}
		problems = append(problems, fmt.Errorf(
			"artifact %s is not a supported release target (nrcc is Linux-only, see ADR 0004)", name))
	}

	return problems
}

// verifyChecksums checks the combined checksum file against the binaries.
func verifyChecksums(dist string, expected map[string]bool) []error {
	var problems []error

	path := filepath.Join(dist, ChecksumFileName)
	// #nosec G304 -- the path is assembled from the dist directory and
	// the fixed checksum file name of this tool.
	content, err := os.ReadFile(path)
	if err != nil {
		return append(problems, fmt.Errorf("read %s: %w", ChecksumFileName, err))
	}

	listed := map[string]string{}
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) != 2 {
			problems = append(problems, fmt.Errorf("%s has a malformed line %q", ChecksumFileName, trimmed))
			continue
		}
		listed[fields[1]] = fields[0]
	}

	for _, name := range sortedNames(expected) {
		sum, ok := listed[name]
		if !ok {
			problems = append(problems, fmt.Errorf("%s does not list %s", ChecksumFileName, name))
			continue
		}
		actual, err := fileChecksum(filepath.Join(dist, name))
		if err != nil {
			problems = append(problems, err)
			continue
		}
		if actual != sum {
			problems = append(problems, fmt.Errorf("%s checksum mismatch for %s: recorded %s, actual %s",
				ChecksumFileName, name, sum, actual))
		}
	}

	for name := range listed {
		if !expected[name] {
			problems = append(problems, fmt.Errorf("%s lists unexpected artifact %s", ChecksumFileName, name))
		}
	}

	return problems
}

// verifySidecars checks the per-binary checksum sidecars.
func verifySidecars(dist string, expected map[string]bool) []error {
	var problems []error

	for _, name := range sortedNames(expected) {
		sidecarPath := filepath.Join(dist, SidecarFileName(name))
		// #nosec G304 -- the path is assembled from the dist directory
		// and the fixed binary names of this tool.
		content, err := os.ReadFile(sidecarPath)
		if err != nil {
			problems = append(problems, fmt.Errorf("read %s: %w", SidecarFileName(name), err))
			continue
		}

		fields := strings.Fields(strings.TrimSpace(string(content)))
		if len(fields) != 2 {
			problems = append(problems, fmt.Errorf("%s has malformed content %q", SidecarFileName(name), strings.TrimSpace(string(content))))
			continue
		}
		if fields[1] != "nrcc" {
			problems = append(problems, fmt.Errorf("%s references %q instead of the normalised name nrcc",
				SidecarFileName(name), fields[1]))
		}

		actual, err := fileChecksum(filepath.Join(dist, name))
		if err != nil {
			problems = append(problems, err)
			continue
		}
		if actual != fields[0] {
			problems = append(problems, fmt.Errorf("%s checksum mismatch for %s: recorded %s, actual %s",
				SidecarFileName(name), name, fields[0], actual))
		}
	}

	return problems
}

// verifyVersionMetadata checks that every binary carries the release
// version.
//
// The linker flag recorded in the build information is the authoritative
// source; the byte scan is only a fallback for binaries whose build
// information is unreadable.
func verifyVersionMetadata(dist string, version Version, targets []Target) []error {
	var problems []error

	for _, target := range targets {
		path := filepath.Join(dist, target.Binary)

		stamped, ok, err := stampedVersion(path)
		if err != nil {
			problems = append(problems, fmt.Errorf("read build information of %s: %w", target.Binary, err))
			continue
		}
		if ok {
			if stamped != version.Semver {
				problems = append(problems, fmt.Errorf("%s carries version %q, expected %q",
					target.Binary, stamped, version.Semver))
			}
			continue
		}

		found, err := containsVersion(path, version.Semver)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		if !found {
			problems = append(problems, fmt.Errorf("%s does not carry the version %q", target.Binary, version.Semver))
		}
	}

	return problems
}

// stampedVersion extracts the version stamped through "-X main.Version=..."
// from the build information of a binary.
//
// The second return value reports whether the build information exposed the
// linker flags at all.
func stampedVersion(path string) (string, bool, error) {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return "", false, err
	}

	for _, setting := range info.Settings {
		if setting.Key != "-ldflags" {
			continue
		}
		value, ok := parseStampedVersion(setting.Value)
		if !ok {
			return "", false, nil
		}
		return value, true, nil
	}

	return "", false, nil
}

// parseStampedVersion extracts the value of "-X main.Version=..." from a
// linker flag string.
//
// The go command records the flag value as it was passed to the linker,
// which includes the quotes of the whole argument.
func parseStampedVersion(flags string) (string, bool) {
	const marker = "-X main.Version="

	trimmed := strings.TrimSpace(flags)
	if len(trimmed) >= 2 && trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
		trimmed = trimmed[1 : len(trimmed)-1]
	}

	idx := strings.Index(trimmed, marker)
	if idx < 0 {
		return "", false
	}

	rest := strings.TrimSpace(trimmed[idx+len(marker):])
	if rest == "" {
		return "", false
	}

	if rest[0] == '"' {
		rest = rest[1:]
		if end := strings.Index(rest, `"`); end >= 0 {
			return rest[:end], true
		}
		return "", false
	}

	if end := strings.IndexAny(rest, " \t"); end >= 0 {
		rest = rest[:end]
	}
	rest = strings.TrimSuffix(rest, `"`)
	if rest == "" {
		return "", false
	}

	return rest, true
}

// containsVersion reports whether the file contains the version string.
func containsVersion(path, version string) (bool, error) {
	// #nosec G304 -- the path is assembled from the dist directory and
	// the fixed binary names of this tool.
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	return bytes.Contains(data, []byte(version)), nil
}

// sortedNames returns the sorted keys of a set.
func sortedNames(set map[string]bool) []string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
