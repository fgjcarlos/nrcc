package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ChecksumFileName is the name of the combined checksum file published
// next to the release binaries.
const ChecksumFileName = "SHA256SUMS"

// ChecksumFileMode is the file mode of the published checksum files.
const ChecksumFileMode os.FileMode = 0o644

// BinaryFileMode is the file mode of the published binaries.
const BinaryFileMode os.FileMode = 0o755

// SidecarFileName returns the name of the per-binary checksum sidecar.
func SidecarFileName(binary string) string {
	return binary + ".sha256"
}

// ldflags returns the linker flags used to stamp the release version into
// the binary.
//
// The build deliberately avoids "-trimpath": the go command drops the
// "-ldflags" build setting from the build information when that flag is
// used, and the release verification reads the stamped version from the
// build information.
func ldflags(version Version) string {
	return fmt.Sprintf("-s -w -X main.Version=%s -extldflags=-static", version.Semver)
}

// runBuild implements the "build" subcommand.
func runBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	version := fs.String("version", "", "release version, for example v1.2.3")
	dist := fs.String("dist", "dist", "directory the artifacts are written to")
	source := fs.String("source", ".", "module root the binaries are built from")

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

	// The dist directory is resolved against the current directory, not
	// against the module root, so the build and the verification always
	// agree on where the artifacts live.
	distDir, err := filepath.Abs(*dist)
	if err != nil {
		return fmt.Errorf("resolve the dist directory: %w", err)
	}

	if err := buildBinaries(*source, distDir, parsed, supportedTargets()); err != nil {
		return err
	}

	return writeChecksums(distDir, supportedTargets())
}

// buildBinaries cross-compiles every supported target into the dist
// directory.
func buildBinaries(source, outputDir string, version Version, targets []Target) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create dist directory: %w", err)
	}

	for _, target := range targets {
		if err := buildBinary(source, outputDir, version, target); err != nil {
			return err
		}
	}

	return nil
}

// buildBinary cross-compiles one target.
//
// The output path is absolute because the build runs inside the module
// root, which is not necessarily the current directory.
func buildBinary(source, outputDir string, version Version, target Target) error {
	output := filepath.Join(outputDir, target.Binary)

	args := []string{
		"build",
		"-tags", "netgo",
		"-buildvcs=false",
		"-ldflags", ldflags(version),
		"-o", output,
		".",
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = source
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS="+target.OS,
		"GOARCH="+target.Arch,
	)
	if target.ARM != "" {
		cmd.Env = append(cmd.Env, "GOARM="+target.ARM)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout

	// #nosec G204 -- the arguments are assembled from the fixed target
	// manifest of this tool, never from user input.
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build %s: %w: %s", target.Platform, err, strings.TrimSpace(stderr.String()))
	}

	// The mode of a produced file depends on the umask of the builder, so
	// it is set explicitly: a published binary must be executable by every
	// consumer of the release.
	if err := os.Chmod(output, BinaryFileMode); err != nil {
		return fmt.Errorf("set the mode of %s: %w", output, err)
	}

	fmt.Printf("built %s -> %s\n", target.Platform, output)
	return nil
}

// writeChecksums writes the combined checksum file and the per-binary
// sidecars.
//
// The sidecar keeps the historical layout ("<hash>  nrcc"): the file name
// in the sidecar is normalised so a verifier can check any binary against
// the same sidecar content.
func writeChecksums(dist string, targets []Target) error {
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.Binary)
	}
	sort.Strings(names)

	combined := &strings.Builder{}
	for _, name := range names {
		sum, err := fileChecksum(filepath.Join(dist, name))
		if err != nil {
			return err
		}
		fmt.Fprintf(combined, "%s  %s\n", sum, name)
	}

	checksumPath := filepath.Join(dist, ChecksumFileName)
	if err := os.WriteFile(checksumPath, []byte(combined.String()), ChecksumFileMode); err != nil {
		return fmt.Errorf("write %s: %w", checksumPath, err)
	}
	if err := os.Chmod(checksumPath, ChecksumFileMode); err != nil {
		return fmt.Errorf("set the mode of %s: %w", checksumPath, err)
	}

	for _, name := range names {
		sum, err := fileChecksum(filepath.Join(dist, name))
		if err != nil {
			return err
		}
		sidecarPath := filepath.Join(dist, SidecarFileName(name))
		content := fmt.Sprintf("%s  nrcc\n", sum)
		if err := os.WriteFile(sidecarPath, []byte(content), ChecksumFileMode); err != nil {
			return fmt.Errorf("write %s: %w", sidecarPath, err)
		}
		if err := os.Chmod(sidecarPath, ChecksumFileMode); err != nil {
			return fmt.Errorf("set the mode of %s: %w", sidecarPath, err)
		}
	}

	return nil
}

// fileChecksum returns the hex encoded SHA-256 digest of a file.
func fileChecksum(path string) (string, error) {
	// #nosec G304 -- the path is assembled from the dist directory and
	// the fixed target manifest of this tool.
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
