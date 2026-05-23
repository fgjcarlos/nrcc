package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const minPnpmMajorVersion = 11

// PackageManager defines operations for package management
type PackageManager interface {
	Install(pkg string) error
	Uninstall(pkg string) error
}

// PnpmPackageManager implements PackageManager using pnpm
type PnpmPackageManager struct {
	WorkDir string
	Bin     string
}

// NewPnpmPackageManager creates a new PnpmPackageManager
func NewPnpmPackageManager(workDir string) *PnpmPackageManager {
	return &PnpmPackageManager{
		WorkDir: workDir,
		Bin:     resolvePnpmBin(),
	}
}

func ensureSupportedPnpm(bin string) error {
	output, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return fmt.Errorf("pnpm is required but was not found or could not run at %q: %w", bin, err)
	}

	version := strings.TrimSpace(string(output))
	majorText, _, _ := strings.Cut(version, ".")
	major, err := strconv.Atoi(majorText)
	if err != nil {
		return fmt.Errorf("could not parse pnpm version %q from %q", version, bin)
	}

	if major < minPnpmMajorVersion {
		return fmt.Errorf("pnpm >= %d is required for safer package operations; found %s at %q", minPnpmMajorVersion, version, bin)
	}

	return nil
}

func resolvePnpmBin() string {
	if bin := os.Getenv("PNPM_BIN"); bin != "" {
		return bin
	}

	if bin, err := exec.LookPath("pnpm"); err == nil {
		return bin
	}

	var candidates []string
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".local", "share", "pnpm", "pnpm"),
			filepath.Join(home, ".npm-global", "bin", "pnpm"),
		)

		if matches, err := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin", "pnpm")); err == nil {
			candidates = append(candidates, matches...)
		}
	}

	candidates = append(candidates,
		"/usr/local/bin/pnpm",
		"/usr/bin/pnpm",
		"/opt/homebrew/bin/pnpm",
	)

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			return candidate
		}
	}

	return "pnpm"
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

// Install installs a package using pnpm add
func (p *PnpmPackageManager) Install(pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	if err := ensureSupportedPnpm(p.Bin); err != nil {
		return err
	}

	cmd := exec.Command(p.Bin, "add", pkg)
	cmd.Dir = p.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install package with pnpm (%s): %w", p.Bin, err)
	}

	return nil
}

// Uninstall uninstalls a package using pnpm remove
func (p *PnpmPackageManager) Uninstall(pkg string) error {
	if err := ValidatePackageName(pkg); err != nil {
		return err
	}
	if err := ensureSupportedPnpm(p.Bin); err != nil {
		return err
	}

	cmd := exec.Command(p.Bin, "remove", pkg)
	cmd.Dir = p.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to uninstall package with pnpm (%s): %w", p.Bin, err)
	}

	return nil
}
