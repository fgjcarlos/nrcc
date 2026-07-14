package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SystemdManager provides an interface for systemd operations.
//
// All exec.Command calls below use the well-known binary paths
// ("systemctl" / "journalctl") directly. Earlier revisions wrapped
// these in a configurable LookPath helper, but those wrappers added
// ceremony to every test (mock setup) without protecting against real
// failures: if systemctl is not on PATH on a supported host, the
// install is broken and the user must fix it. The current shape
// keeps the code simple; users with custom systemd paths can
// override the binary lookup at the DockerService layer if needed.
type SystemdManager interface {
	IsAvailable() bool
	DaemonReload() error
	EnableAndStart(unit string) error
	Stop(unit string) error
	Disable(unit string) error
	GetServiceStatus(unit string) (string, error) // returns: active, inactive, failed, unknown
}

// execSystemdManager implements SystemdManager using exec.Command
type execSystemdManager struct{}

// NewSystemdManager creates a new systemd manager that uses exec.Command
func NewSystemdManager() SystemdManager {
	return &execSystemdManager{}
}

// IsAvailable checks if systemd is available on the system
func (m *execSystemdManager) IsAvailable() bool {
	// Standard systemd marker directory
	_, err := os.Stat("/run/systemd/system")
	return err == nil
}

// DaemonReload runs systemctl daemon-reload
func (m *execSystemdManager) DaemonReload() error {
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl daemon-reload failed: %w", err)
	}
	return nil
}

// EnableAndStart runs systemctl enable --now <unit>
func (m *execSystemdManager) EnableAndStart(unit string) error {
	cmd := exec.Command("systemctl", "enable", "--now", unit)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl enable --now %s failed: %w", unit, err)
	}
	return nil
}

// Stop runs systemctl stop <unit> (non-fatal if unit not found)
func (m *execSystemdManager) Stop(unit string) error {
	cmd := exec.Command("systemctl", "stop", unit)
	if err := cmd.Run(); err != nil {
		// Exit code 5 means unit not loaded (already stopped) — treat as success
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 5 {
			return nil
		}
		return fmt.Errorf("systemctl stop %s failed: %w", unit, err)
	}
	return nil
}

// Disable runs systemctl disable <unit> (non-fatal if unit not found)
func (m *execSystemdManager) Disable(unit string) error {
	cmd := exec.Command("systemctl", "disable", unit)
	if err := cmd.Run(); err != nil {
		// Exit code 5 means unit not loaded (already disabled) — treat as success
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 5 {
			return nil
		}
		return fmt.Errorf("systemctl disable %s failed: %w", unit, err)
	}
	return nil
}

// GetServiceStatus returns the active status of a service
func (m *execSystemdManager) GetServiceStatus(unit string) (string, error) {
	cmd := exec.Command("systemctl", "is-active", unit)
	output, err := cmd.Output()
	if err != nil {
		// If the service doesn't exist, is-active returns non-zero
		// But we can still get meaningful output
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() != 0 {
			status := strings.TrimSpace(string(output))
			if status == "inactive" {
				return "inactive", nil
			}
			if status == "failed" {
				return "failed", nil
			}
			return "unknown", nil
		}
		return "unknown", fmt.Errorf("failed to get service status: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
