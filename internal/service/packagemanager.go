package service

import (
	"fmt"
	"os"
	"os/exec"
)

// PackageManager defines operations for package management
type PackageManager interface {
	Install(pkg string) error
	Uninstall(pkg string) error
}

// PnpmPackageManager implements PackageManager using pnpm
type PnpmPackageManager struct {
	WorkDir string
}

// NewPnpmPackageManager creates a new PnpmPackageManager
func NewPnpmPackageManager(workDir string) *PnpmPackageManager {
	return &PnpmPackageManager{
		WorkDir: workDir,
	}
}

// Install installs a package using pnpm add
func (p *PnpmPackageManager) Install(pkg string) error {
	cmd := exec.Command("pnpm", "add", pkg)
	cmd.Dir = p.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}

	return nil
}

// Uninstall uninstalls a package using pnpm remove
func (p *PnpmPackageManager) Uninstall(pkg string) error {
	cmd := exec.Command("pnpm", "remove", pkg)
	cmd.Dir = p.WorkDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to uninstall package: %w", err)
	}

	return nil
}
