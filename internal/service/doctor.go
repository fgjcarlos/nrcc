package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

// DoctorService runs health checks on the Node-RED Control Center
type DoctorService struct {
	dataDir        string
	processManager *ProcessManager
	logService     *LogService
	localAccess    *LocalAccessService
}

// NewDoctorService creates a new DoctorService
func NewDoctorService(dataDir string) *DoctorService {
	return &DoctorService{
		dataDir: dataDir,
	}
}

// SetProcessManager injects the ProcessManager (nil-safe)
func (d *DoctorService) SetProcessManager(pm *ProcessManager) {
	d.processManager = pm
}

// SetLogService injects the LogService (nil-safe)
func (d *DoctorService) SetLogService(ls *LogService) {
	d.logService = ls
}

// SetLocalAccessService injects the LocalAccessService (nil-safe)
func (d *DoctorService) SetLocalAccessService(access *LocalAccessService) {
	d.localAccess = access
}

// Run executes all checks and returns a DoctorReport
func (d *DoctorService) Run(ctx context.Context) model.DoctorReport {
	report := model.DoctorReport{
		GeneratedAt: time.Now().UTC(),
		Checks:      []model.DoctorCheck{},
	}

	// Define all checks to run
	checks := []func(context.Context) model.DoctorCheck{
		d.checkNodeRedInstalled,
		d.checkNodeVersion,
		d.checkNpmVersion,
		d.checkDataDirWritable,
		d.checkUserdirExists,
		d.checkSettingsFile,
		d.checkFlowsFile,
		d.checkProcessRunning,
		d.checkPortAvailable,
		d.checkLocalAccess,
		d.checkLogDirWritable,
		d.checkDbAccessible,
		d.checkDiskSpace,
		d.checkNrccVersion,
	}

	// Run all checks
	for _, checkFn := range checks {
		// Create a timeout context for each check (5 seconds)
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		check := runCheckSafely(checkCtx, checkFn)
		cancel()

		report.Checks = append(report.Checks, check)
	}

	// Calculate overall status
	report.OverallStatus = calculateOverallStatus(report.Checks)

	return report
}

// runCheckSafely executes a check and catches panics.
// On panic it returns a DoctorCheck with Status="fail" and the panic message
// captured in Message so the caller always gets a meaningful entry.
func runCheckSafely(ctx context.Context, checkFn func(context.Context) model.DoctorCheck) (result model.DoctorCheck) {
	defer func() {
		if r := recover(); r != nil {
			result = model.DoctorCheck{
				ID:      "unknown",
				Label:   "Unknown Check",
				Status:  model.CheckStatusFail,
				Message: fmt.Sprintf("check panicked: %v", r),
			}
		}
	}()

	result = checkFn(ctx)
	return
}

// Check 1: node-red-installed
func (d *DoctorService) checkNodeRedInstalled(ctx context.Context) model.DoctorCheck {
	name := "node-red-installed"

	// Check common locations
	locations := []string{
		"/usr/local/bin/node-red",
		filepath.Join(os.Getenv("HOME"), ".npm", "bin", "node-red"),
		filepath.Join(d.dataDir, "node_modules", "node-red", "red.js"),
	}

	for _, loc := range locations {
		if platform.Exists(loc) {
			return model.DoctorCheck{
				ID:      name,
				Label:   "Node-RED Installed",
				Status:  model.CheckStatusPass,
				Message: fmt.Sprintf("Node-RED found at %s", loc),
			}
		}
	}

	// Also check if node-red is in PATH
	if _, err := exec.LookPath("node-red"); err == nil {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Node-RED Installed",
			Status:  model.CheckStatusPass,
			Message: "Node-RED found in PATH",
		}
	}

	return model.DoctorCheck{
		ID:      name,
		Label:   "Node-RED Installed",
		Status:  model.CheckStatusFail,
		Message: "Node-RED not found in PATH or common locations",
	}
}

// Check 2: node-version
func (d *DoctorService) checkNodeVersion(ctx context.Context) model.DoctorCheck {
	name := "node-version"

	output, err := exec.CommandContext(ctx, "node", "--version").Output()
	if err != nil {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Node Version",
			Status:  model.CheckStatusFail,
			Message: "Node.js not found",
		}
	}

	version := strings.TrimSpace(string(output))
	// Parse version (e.g., "v18.12.0" -> 18)
	major := extractMajorVersion(version)

	if major < 18 {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Node Version",
			Status:  model.CheckStatusWarn,
			Message: fmt.Sprintf("Node version %s is below recommended 18.x", version),
		}
	}

	return model.DoctorCheck{
		ID:      name,
		Label:   "Node Version",
		Status:  model.CheckStatusPass,
		Message: fmt.Sprintf("Node version %s is compatible", version),
	}
}

// Check 3: npm-version
func (d *DoctorService) checkNpmVersion(ctx context.Context) model.DoctorCheck {
	name := "npm-version"

	output, err := exec.CommandContext(ctx, "npm", "--version").Output()
	if err != nil {
		return model.DoctorCheck{
			ID:      name,
			Label:   "npm Version",
			Status:  model.CheckStatusFail,
			Message: "npm not found",
		}
	}

	version := strings.TrimSpace(string(output))
	return model.DoctorCheck{
		ID:      name,
		Label:   "npm Version",
		Status:  model.CheckStatusPass,
		Message: fmt.Sprintf("npm version %s is installed", version),
	}
}

// Check 4: data-dir-writable
func (d *DoctorService) checkDataDirWritable(ctx context.Context) model.DoctorCheck {
	name := "data-dir-writable"

	if !platform.Exists(d.dataDir) {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Data Directory Writable",
			Status:  model.CheckStatusFail,
			Message: fmt.Sprintf("Data directory does not exist: %s", d.dataDir),
		}
	}

	// Test write capability
	testFile := filepath.Join(d.dataDir, ".write_test")
	err := os.WriteFile(testFile, []byte("test"), 0600)
	defer os.Remove(testFile)

	if err != nil {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Data Directory Writable",
			Status:  model.CheckStatusFail,
			Message: fmt.Sprintf("Data directory is not writable: %v", err),
		}
	}

	return model.DoctorCheck{
		ID:      name,
		Label:   "Data Directory Writable",
		Status:  model.CheckStatusPass,
		Message: fmt.Sprintf("Data directory is writable: %s", d.dataDir),
	}
}

// Check 5: userdir-exists
func (d *DoctorService) checkUserdirExists(ctx context.Context) model.DoctorCheck {
	name := "userdir-exists"

	userDir := filepath.Join(d.dataDir, "nodered")
	if !platform.Exists(userDir) {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Node-RED User Directory",
			Status:  model.CheckStatusFail,
			Message: fmt.Sprintf("Node-RED user directory does not exist: %s", userDir),
		}
	}

	return model.DoctorCheck{
		ID:      name,
		Label:   "Node-RED User Directory",
		Status:  model.CheckStatusPass,
		Message: fmt.Sprintf("Node-RED user directory exists: %s", userDir),
	}
}

// Check 6: settings-file
func (d *DoctorService) checkSettingsFile(ctx context.Context) model.DoctorCheck {
	name := "settings-file"

	userDir := filepath.Join(d.dataDir, "nodered")
	settingsPath := filepath.Join(userDir, "settings.js")

	if !platform.Exists(settingsPath) {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Settings File",
			Status:  model.CheckStatusFail,
			Message: fmt.Sprintf("settings.js not found in %s", userDir),
		}
	}

	return model.DoctorCheck{
		ID:      name,
		Label:   "Settings File",
		Status:  model.CheckStatusPass,
		Message: "settings.js exists",
	}
}

// Check 7: flows-file
func (d *DoctorService) checkFlowsFile(ctx context.Context) model.DoctorCheck {
	name := "flows-file"

	userDir := filepath.Join(d.dataDir, "nodered")
	hostname, _ := os.Hostname()

	// Check for flows.json or flows_<hostname>.json
	paths := []string{
		filepath.Join(userDir, "flows.json"),
		filepath.Join(userDir, fmt.Sprintf("flows_%s.json", hostname)),
	}

	for _, path := range paths {
		if platform.Exists(path) {
			return model.DoctorCheck{
				ID:      name,
				Label:   "Flows File",
				Status:  model.CheckStatusPass,
				Message: fmt.Sprintf("Flows file exists: %s", filepath.Base(path)),
			}
		}
	}

	return model.DoctorCheck{
		ID:      name,
		Label:   "Flows File",
		Status:  model.CheckStatusWarn,
		Message: "No flows.json file found",
	}
}

// Check 8: process-running
func (d *DoctorService) checkProcessRunning(ctx context.Context) model.DoctorCheck {
	name := "process-running"

	if d.processManager == nil {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Node-RED Process Running",
			Status:  model.CheckStatusWarn,
			Message: "Process manager not available",
		}
	}

	status := d.processManager.Status()
	if status.Running {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Node-RED Process Running",
			Status:  model.CheckStatusPass,
			Message: fmt.Sprintf("Node-RED is running (PID: %d)", status.PID),
		}
	}

	return model.DoctorCheck{
		ID:      name,
		Label:   "Node-RED Process Running",
		Status:  model.CheckStatusWarn,
		Message: "Node-RED is not running",
	}
}

// Check 9: port-available
func (d *DoctorService) checkPortAvailable(ctx context.Context) model.DoctorCheck {
	name := "port-available"

	address := "127.0.0.1:1880"
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err == nil {
		conn.Close()
		return model.DoctorCheck{
			ID:      name,
			Label:   "Node-RED Port Available",
			Status:  model.CheckStatusPass,
			Message: "Port 1880 is open and accessible",
		}
	}

	return model.DoctorCheck{
		ID:      name,
		Label:   "Node-RED Port Available",
		Status:  model.CheckStatusWarn,
		Message: "Port 1880 is not accessible",
	}
}

// Check 10: local-access
func (d *DoctorService) checkLocalAccess(ctx context.Context) model.DoctorCheck {
	_ = ctx
	name := "local-access"

	if d.localAccess == nil {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Stable Local Access",
			Status:  model.CheckStatusWarn,
			Message: "Local access integration is not initialized",
		}
	}

	status := d.localAccess.Status()
	if status.Configured && status.Operational {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Stable Local Access",
			Status:  model.CheckStatusPass,
			Message: fmt.Sprintf("Stable hostname available at %s", status.URL),
		}
	}

	return model.DoctorCheck{
		ID:      name,
		Label:   "Stable Local Access",
		Status:  model.CheckStatusWarn,
		Message: status.Message,
	}
}

// Check 10: log-dir-writable
func (d *DoctorService) checkLogDirWritable(ctx context.Context) model.DoctorCheck {
	name := "log-dir-writable"

	logsDir := filepath.Join(d.dataDir, "logs")
	if !platform.Exists(logsDir) {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Log Directory Writable",
			Status:  model.CheckStatusFail,
			Message: fmt.Sprintf("Logs directory does not exist: %s", logsDir),
		}
	}

	// Test write capability
	testFile := filepath.Join(logsDir, ".write_test")
	err := os.WriteFile(testFile, []byte("test"), 0600)
	defer os.Remove(testFile)

	if err != nil {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Log Directory Writable",
			Status:  model.CheckStatusFail,
			Message: fmt.Sprintf("Logs directory is not writable: %v", err),
		}
	}

	return model.DoctorCheck{
		ID:      name,
		Label:   "Log Directory Writable",
		Status:  model.CheckStatusPass,
		Message: fmt.Sprintf("Logs directory is writable: %s", logsDir),
	}
}

// Check 11: db-accessible
func (d *DoctorService) checkDbAccessible(ctx context.Context) model.DoctorCheck {
	name := "db-accessible"

	dbPath := filepath.Join(d.dataDir, "nrcc.db")
	if !platform.Exists(dbPath) {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Database Accessible",
			Status:  model.CheckStatusWarn,
			Message: fmt.Sprintf("Database file not found: %s", dbPath),
		}
	}

	return model.DoctorCheck{
		ID:      name,
		Label:   "Database Accessible",
		Status:  model.CheckStatusPass,
		Message: "Database file exists",
	}
}

// Check 12: disk-space
func (d *DoctorService) checkDiskSpace(ctx context.Context) model.DoctorCheck {
	name := "disk-space"

	var stat syscall.Statfs_t
	err := syscall.Statfs(d.dataDir, &stat)
	if err != nil {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Disk Space",
			Status:  model.CheckStatusWarn,
			Message: fmt.Sprintf("Could not check disk space: %v", err),
		}
	}

	// Available blocks * block size
	availableBytes := int64(stat.Bavail) * stat.Bsize
	const (
		minFailBytes = 100 * 1024 * 1024 // 100MB
		minWarnBytes = 500 * 1024 * 1024 // 500MB
	)

	if availableBytes < minFailBytes {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Disk Space",
			Status:  model.CheckStatusFail,
			Message: fmt.Sprintf("Critical: Only %.0f MB available", float64(availableBytes)/1024/1024),
		}
	}

	if availableBytes < minWarnBytes {
		return model.DoctorCheck{
			ID:      name,
			Label:   "Disk Space",
			Status:  model.CheckStatusWarn,
			Message: fmt.Sprintf("Low disk space: %.0f MB available", float64(availableBytes)/1024/1024),
		}
	}

	return model.DoctorCheck{
		ID:      name,
		Label:   "Disk Space",
		Status:  model.CheckStatusPass,
		Message: fmt.Sprintf("Disk space OK: %.0f MB available", float64(availableBytes)/1024/1024),
	}
}

// Check 13: nrcc-version
func (d *DoctorService) checkNrccVersion(ctx context.Context) model.DoctorCheck {
	name := "nrcc-version"

	version := "dev"
	return model.DoctorCheck{
		ID:      name,
		Label:   "NRCC Version",
		Status:  model.CheckStatusPass,
		Message: fmt.Sprintf("NRCC version: %s", version),
	}
}

// SaveReport persists a DoctorReport to the database
func (d *DoctorService) SaveReport(db *sql.DB, report model.DoctorReport) error {
	reportID := fmt.Sprintf("doctor_%d_%s", time.Now().UnixNano(), randomID(8))

	checksJSON, err := json.Marshal(report.Checks)
	if err != nil {
		return fmt.Errorf("marshal checks: %w", err)
	}

	query := `
		INSERT INTO doctor_runs (id, generated_at, overall_status, checks_json, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = db.Exec(query, reportID, report.GeneratedAt, report.OverallStatus, string(checksJSON), time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert doctor report: %w", err)
	}

	return nil
}

// Helper: extract major version from version string (e.g., "v18.12.0" -> 18)
func extractMajorVersion(version string) int {
	version = strings.TrimPrefix(version, "v")
	parts := strings.Split(version, ".")
	if len(parts) > 0 {
		var major int
		fmt.Sscanf(parts[0], "%d", &major)
		return major
	}
	return 0
}

// Helper: calculate overall status based on check results
func calculateOverallStatus(checks []model.DoctorCheck) string {
	hasFail := false
	hasWarn := false

	for _, check := range checks {
		if check.Status == model.CheckStatusFail {
			hasFail = true
		} else if check.Status == model.CheckStatusWarn {
			hasWarn = true
		}
	}

	if hasFail {
		return model.OverallCritical
	}
	if hasWarn {
		return model.OverallDegraded
	}
	return model.OverallHealthy
}
