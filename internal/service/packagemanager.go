package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// PackageManager defines operations for package management
type PackageManager interface {
	Install(pkg string) error
	Uninstall(pkg string) error
}

// NpmPackageManager implements PackageManager using npm
type NpmPackageManager struct {
	WorkDir string
	Bin     string
}

// NewNpmPackageManager creates a new NpmPackageManager, resolving the npm binary path.
func NewNpmPackageManager(workDir string) *NpmPackageManager {
	return &NpmPackageManager{
		WorkDir: workDir,
		Bin:     resolveNpmBin(),
	}
}

// ensureNpm runs bin --version to verify that npm is available and executable.
// If the binary cannot be executed, it returns a descriptive error.
func ensureNpm(bin string) error {
	if err := exec.Command(bin, "--version").Run(); err != nil {
		return fmt.Errorf("npm is required but was not found or could not run at %q: %w", bin, err)
	}
	return nil
}

// resolveNpmBin returns the path to the npm binary using the following precedence:
//  1. NPM_BIN environment variable (set by /etc/nrcc/nrcc.env via issue #257)
//  2. npm on PATH via exec.LookPath
//  3. Known installation candidate paths (nvm, system, Homebrew)
//  4. Bare "npm" (falls back to PATH resolution at exec time)
func resolveNpmBin() string {
	if bin := os.Getenv("NPM_BIN"); bin != "" {
		return bin
	}

	if bin, err := exec.LookPath("npm"); err == nil {
		return bin
	}

	var candidates []string
	if home, err := os.UserHomeDir(); err == nil {
		if matches, err := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin", "npm")); err == nil {
			candidates = append(candidates, matches...)
		}
	}

	candidates = append(candidates,
		"/usr/local/bin/npm",
		"/usr/bin/npm",
		"/opt/homebrew/bin/npm",
	)

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			return candidate
		}
	}

	return "npm"
}

var (
	ErrInvalidPackageName = errors.New("invalid package name")

	// Matches: pkg, @scope/pkg, pkg@version, @scope/pkg@^1.2.3
	validPackageRe = regexp.MustCompile(`^(@[a-z0-9][\w.\-]*/)?[a-z0-9][\w.\-]*(@[^\s]+)?$`)
)

// ValidatePackageName checks that a string is a safe npm package specifier.
func ValidatePackageName(pkg string) error {
	if pkg == "" {
		return fmt.Errorf("%w: empty name", ErrInvalidPackageName)
	}
	if len(pkg) > 214 {
		return fmt.Errorf("%w: exceeds max length", ErrInvalidPackageName)
	}
	if strings.ContainsAny(pkg, ";|&$`(){}!'\"\\") {
		return fmt.Errorf("%w: contains shell metacharacters", ErrInvalidPackageName)
	}
	if strings.Contains(pkg, "..") || strings.HasPrefix(pkg, "/") || strings.HasPrefix(pkg, ".") {
		return fmt.Errorf("%w: path-like specifier not allowed", ErrInvalidPackageName)
	}
	if strings.Contains(pkg, "://") {
		return fmt.Errorf("%w: URL specifier not allowed", ErrInvalidPackageName)
	}
	if strings.ContainsAny(pkg, " \t\n\r") {
		return fmt.Errorf("%w: contains whitespace", ErrInvalidPackageName)
	}
	if !validPackageRe.MatchString(pkg) {
		return fmt.Errorf("%w: %q does not match npm naming rules", ErrInvalidPackageName, pkg)
	}
	return nil
}

// Install installs a package using npm install
func (p *NpmPackageManager) Install(pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	if err := ensureNpm(p.Bin); err != nil {
		return err
	}

	cmd := exec.Command(p.Bin, "install", "--no-fund", "--no-audit", pkg)
	cmd.Dir = p.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install package with npm (%s): %w", p.Bin, err)
	}

	return nil
}

// Uninstall uninstalls a package using npm uninstall
func (p *NpmPackageManager) Uninstall(pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	if err := ensureNpm(p.Bin); err != nil {
		return err
	}

	cmd := exec.Command(p.Bin, "uninstall", "--no-fund", "--no-audit", pkg)
	cmd.Dir = p.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to uninstall package with npm (%s): %w", p.Bin, err)
	}

	return nil
}
