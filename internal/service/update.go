package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/store"
)

// execRunner abstracts subprocess execution for testability.
// This allows tests to mock npm calls without actually invoking npm.
type execRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type defaultRunner struct{}

func (r *defaultRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// backupCreator is the minimal slice of BackupService that the update flow needs
// to take a real pre-update backup. It is an interface so tests can inject a
// fake and so UpdateService does not depend on the full BackupService surface.
type backupCreator interface {
	CreateTyped(backupType model.BackupType, name string) (model.Backup, error)
}

// UpdateService handles Node-RED update detection and caching.
//
// Design: Backend-owned polling goroutine runs on a configurable interval (default 4 hours),
// checking for new Node-RED versions via npm. Results are cached in-memory (for fast reads)
// and persisted to ./data/update_cache.json (for durability across restarts).
//
// The service provides two access patterns:
// 1. GetCachedStatus(): Hot-path read (sub-millisecond) — returns last-known status
// 2. ForceCheck(): Fresh check — spawns an npm call, blocks up to timeout, updates cache
//
// Concurrency: Mutex-guarded in-memory cache + atomic JSONStore writes ensure thread-safety.
// Deduplication: checkInProgress TryLock prevents concurrent npm calls during manual checks.
//
// Resilience: Version detection falls back to "unknown" rather than failing hard, error messages
// are sanitized to avoid exposing raw npm exit codes to the UI, and cache persists across restarts.
//
// Multi-stage Update Flow:
// - State machine tracks: Idle → BackingUp → Applying → Completed/Failed
// - applyMu guards concurrent updates; only one update flow at a time
// - flowState stores current operation state (thread-safe via flowStateMu)
// - backupStore persists backup metadata to ./data/update_backups.json (max 5 entries)
type UpdateService struct {
	dataDir               string
	cache                 model.UpdateCacheEntry
	cacheMu               sync.RWMutex
	store                 *store.JSONStore[model.UpdateCacheEntry]
	stopChan              chan struct{}
	runner                execRunner
	pollInterval          time.Duration
	npmTimeout            time.Duration
	checkInProgress       sync.Mutex
	getInstalledVersionFn func(ctx context.Context) string
	getLatestVersionFn    func(ctx context.Context) (string, error)
	// New fields for update flow state machine
	applyMu       sync.Mutex
	flowState     model.UpdateFlowState
	flowStateMu   sync.RWMutex
	backupStore   *store.JSONStore[[]model.BackupEntry]
	backupSvc     backupCreator
}

// SetBackupCreator wires the backup engine used to take a real archive before an
// update is applied. It must be called during server wiring; without it, the
// update flow refuses to proceed (it will not fabricate a phantom backup).
func (s *UpdateService) SetBackupCreator(bc backupCreator) {
	s.backupSvc = bc
}

const (
	defaultPollInterval = 4 * time.Hour
	defaultNPMTimeout   = 10 * time.Second
	updateCacheFile     = "update_cache.json"
	updateBackupsFile   = "update_backups.json"
)

// NewUpdateService creates a new update service
func NewUpdateService(dataDir string) *UpdateService {
	s := &UpdateService{
		dataDir:      dataDir,
		stopChan:     make(chan struct{}),
		runner:       &defaultRunner{},
		pollInterval: defaultPollInterval,
		npmTimeout:   defaultNPMTimeout,
		flowState:    model.UpdateFlowState{State: model.StateIdle, Phase: "idle"},
	}

	s.store = store.NewJSONStore[model.UpdateCacheEntry](
		fmt.Sprintf("%s/%s", dataDir, updateCacheFile),
	)
	
	s.backupStore = store.NewJSONStore[[]model.BackupEntry](
		fmt.Sprintf("%s/%s", dataDir, updateBackupsFile),
	)

	// Set default version checking functions
	s.getInstalledVersionFn = s.getInstalledVersionInternal
	s.getLatestVersionFn = s.getLatestVersionInternal

	// Load cache from disk if it exists
	if s.store.Exists() {
		if cached, err := s.store.Read(); err == nil {
			s.cacheMu.Lock()
			s.cache = cached
			s.cacheMu.Unlock()
		}
	}
	
	// On startup, if cache is still empty, perform an initial check synchronously.
	// This ensures the UI doesn't show "unknown" on first page load.
	s.cacheMu.Lock()
	cacheEmpty := s.cache.CurrentVersion == "" && s.cache.LatestVersion == ""
	s.cacheMu.Unlock()
	
	if cacheEmpty {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		s.performCheck(ctx)
		cancel()
	}

	return s
}

// Start begins the background polling goroutine.
// The goroutine exits when ctx is cancelled.
func (s *UpdateService) Start(ctx context.Context) {
	go s.pollingLoop(ctx)
}

// Stop cancels the polling goroutine and waits for it to exit.
func (s *UpdateService) Stop() {
	close(s.stopChan)
}

// pollingLoop runs the background polling cycle.
// Runs once per pollInterval (default 4 hours) unless ctx is cancelled or Stop() is called.
// Each iteration performs a fresh npm check and persists the result to disk.
// Errors are logged but don't stop the loop — polling continues on the next interval.
func (s *UpdateService) pollingLoop(ctx context.Context) {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.performCheck(ctx)
		}
	}
}

// performCheck executes an update check and updates the cache.
// Called once per pollInterval by pollingLoop().
// Updates both in-memory cache (under mutex) and disk file (atomically via JSONStore).
// Errors in npm calls are captured in the cache Error field; cache is updated regardless.
func (s *UpdateService) performCheck(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, s.npmTimeout)
	defer cancel()

	entry := s.performCheckInternal(checkCtx)
	entry.CheckedAt = time.Now().UTC()

	s.cacheMu.Lock()
	s.cache = entry
	s.cacheMu.Unlock()

	// Write to disk atomically
	_ = s.store.Write(entry)
}

// performCheckInternal does the actual npm check logic.
// Runs `npm list -g node-red --json` to get installed version.
// Runs `npm view node-red version` to get latest available version.
// Compares versions and updates the UpdateAvailable flag.
// Returns immediately if npm calls fail; error is captured in the result with a sanitized message.
func (s *UpdateService) performCheckInternal(ctx context.Context) model.UpdateCacheEntry {
	entry := model.UpdateCacheEntry{
		UpdateAvailable: false,
	}

	// Get current version
	currentVersion := s.getInstalledVersionFn(ctx)
	entry.CurrentVersion = currentVersion

	// Get latest version from npm
	latestVersion, err := s.getLatestVersionFn(ctx)
	if err != nil {
		entry.Error = s.sanitizeErrorMessage(err)
		entry.LatestVersion = ""
		return entry
	}

	entry.LatestVersion = latestVersion
	entry.Error = ""

	// Compare versions
	if s.compareVersions(currentVersion, latestVersion) < 0 {
		entry.UpdateAvailable = true
	}

	return entry
}

// GetCachedStatus returns the in-memory cache (zero-alloc hot path)
func (s *UpdateService) GetCachedStatus() model.UpdateCacheEntry {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return s.cache
}

// ForceCheck runs an immediate npm check, updates the cache, and returns the result.
// Blocks until the check completes (up to npmTimeout, typically 10 seconds).
// If a check is already in progress (concurrent manual checks), returns current status without spawning a new npm call.
// Used by the frontend /api/updates/check endpoint for manual "Check Now" button clicks.
func (s *UpdateService) ForceCheck(ctx context.Context) (model.UpdateCacheEntry, error) {
	// Try to acquire the check lock without blocking
	acquired := s.checkInProgress.TryLock()
	if !acquired {
		// A check is already in progress; return current cache
		return s.GetCachedStatus(), nil
	}
	defer s.checkInProgress.Unlock()

	// Perform the check with a timeout
	checkCtx, cancel := context.WithTimeout(ctx, s.npmTimeout)
	defer cancel()

	entry := s.performCheckInternal(checkCtx)
	entry.CheckedAt = time.Now().UTC()

	// Update cache
	s.cacheMu.Lock()
	s.cache = entry
	s.cacheMu.Unlock()

	// Write to disk atomically
	if err := s.store.Write(entry); err != nil {
		// Log but don't fail; cache is still updated in-memory
		fmt.Printf("warning: failed to write cache to disk: %v\n", err)
	}

	return entry, nil
}

// CheckForUpdate checks if an update is available for Node-RED.
// This method is retained for backward compatibility with cmd/ CLI usage.
func (s *UpdateService) CheckForUpdate() (model.UpdateStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.npmTimeout)
	defer cancel()

	entry := s.performCheckInternal(ctx)

	return model.UpdateStatus{
		CurrentVersion:  entry.CurrentVersion,
		LatestVersion:   entry.LatestVersion,
		UpdateAvailable: entry.UpdateAvailable,
	}, nil
}

// GetFlowState returns a deep copy of the current update flow state.
// Thread-safe: acquires read lock before reading flowState.
func (s *UpdateService) GetFlowState() model.UpdateFlowState {
	s.flowStateMu.RLock()
	defer s.flowStateMu.RUnlock()
	
	// Include current availableVersion from cache
	state := s.flowState
	cached := s.GetCachedStatus()
	if cached.UpdateAvailable {
		state.AvailableVersion = cached.LatestVersion
	}
	
	return state
}

// setFlowState updates flowState with write lock.
// Internal method; should only be called from ApplyUpdateWithBackup.
func (s *UpdateService) setFlowState(state model.UpdateFlowState) {
	s.flowStateMu.Lock()
	defer s.flowStateMu.Unlock()
	s.flowState = state
}

// SetFlowState updates flowState (public for testing).
// Thread-safe: acquires write lock before updating flowState.
// NOTE: Used primarily in integration tests to simulate state progression.
func (s *UpdateService) SetFlowState(state model.UpdateFlowState) {
	s.flowStateMu.Lock()
	defer s.flowStateMu.Unlock()
	s.flowState = state
}

// CreateBackup takes a real archive of the Node-RED data directory before an
// update is applied, delegating to the configured backup engine. It returns a
// BackupEntry referencing the on-disk archive (with its actual size), or an
// error if no backup could be written — the caller MUST abort the update in
// that case so we never apply an update without a restore point.
//
// Does NOT persist the entry to the update catalog; the caller decides
// persistence (see AppendBackup / ApplyUpdateWithBackup).
func (s *UpdateService) CreateBackup(ctx context.Context, fromVersion string) (model.BackupEntry, error) {
	if s.backupSvc == nil {
		return model.BackupEntry{}, fmt.Errorf("backup engine not configured: refusing to apply update without a real backup")
	}

	name := fmt.Sprintf("Pre-update backup (from %s)", fromVersion)
	backup, err := s.backupSvc.CreateTyped(model.BackupTypeManual, name)
	if err != nil {
		return model.BackupEntry{}, fmt.Errorf("failed to create pre-update backup: %w", err)
	}

	return model.BackupEntry{
		ID:          backup.ID,
		Path:        backup.Path,
		SizeBytes:   backup.SizeBytes,
		Timestamp:   time.Now().UTC(),
		FromVersion: fromVersion,
		Status:      "completed",
	}, nil
}

// History returns the applied-update backup catalog (most recent updates), or an
// empty slice when none exist. This is the real source for the update history
// endpoint.
func (s *UpdateService) History() []model.BackupEntry {
	if !s.backupStore.Exists() {
		return []model.BackupEntry{}
	}
	entries, err := s.backupStore.Read()
	if err != nil {
		return []model.BackupEntry{}
	}
	return entries
}

// AppendBackup persists a backup entry to the catalog, keeping max 5 entries.
// If catalog already has 5 entries, removes the oldest before appending the new one.
// Thread-safe via backupStore mutex.
func (s *UpdateService) AppendBackup(entry model.BackupEntry) error {
	// Ensure the data directory exists (parent of update_backups.json)
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	
	// Read current backups
	backups := []model.BackupEntry{}
	if s.backupStore.Exists() {
		if read, err := s.backupStore.Read(); err == nil {
			backups = read
		}
	}
	
	// Keep max 5; remove oldest if at limit
	if len(backups) >= 5 {
		backups = backups[1:] // Remove first (oldest by append order)
	}
	
	// Append new entry
	backups = append(backups, entry)
	
	// Write back to store
	return s.backupStore.Write(backups)
}

// ApplyUpdateWithBackup orchestrates the full update flow: BackingUp → Applying → Completed/Failed.
// Only one update can proceed at a time; returns error if state != Idle.
// Flow:
//   1. Acquire applyMu lock
//   2. Check flowState.State == StateIdle; return error if not
//   3. Set state = BackingUp
//   4. Call CreateBackup; persist to catalog
//   5. Set state = Applying
//   6. Call ApplyUpdate (npm install)
//   7. Set state = Completed or Failed based on npm result
//   8. Release applyMu lock
//
// Backend operation is asynchronous (logs output but does not block caller).
// Frontend polls /api/updates/state to track progress.
func (s *UpdateService) ApplyUpdateWithBackup(ctx context.Context) error {
	// Acquire lock for exclusive update flow
	acquired := s.applyMu.TryLock()
	if !acquired {
		// Another update is in progress
		return fmt.Errorf("update already in progress")
	}
	defer s.applyMu.Unlock()
	
	// Check if we can start: state must be Idle
	s.flowStateMu.RLock()
	if s.flowState.State != model.StateIdle {
		s.flowStateMu.RUnlock()
		return fmt.Errorf("update cannot start: state is %s", s.flowState.State)
	}
	s.flowStateMu.RUnlock()
	
	// Get current version for backup record
	preApplyStatus := s.GetCachedStatus()
	fromVersion := preApplyStatus.CurrentVersion
	if fromVersion == "" {
		fromVersion = "unknown"
	}
	
	// Step 1: BackingUp
	s.setFlowState(model.UpdateFlowState{
		State: model.StateBackingUp,
		Phase: "backup",
	})
	
	// Create backup
	backupEntry, err := s.CreateBackup(ctx, fromVersion)
	if err != nil {
		s.setFlowState(model.UpdateFlowState{
			State: model.StateFailed,
			Error: "backup_failed",
			Phase: "backup",
		})
		return err
	}
	
	// Persist backup to catalog
	if err := s.AppendBackup(backupEntry); err != nil {
		// Log error but continue (backup creation succeeded, just persistence failed)
		fmt.Printf("warning: failed to persist backup catalog: %v\n", err)
	}
	
	// Step 2: Applying
	s.setFlowState(model.UpdateFlowState{
		State:            model.StateApplying,
		Phase:            "applying",
		BackupID:         backupEntry.ID,
		AvailableVersion: preApplyStatus.LatestVersion,
	})
	
	// Execute npm update
	err = s.ApplyUpdate()
	if err != nil {
		s.setFlowState(model.UpdateFlowState{
			State:    model.StateFailed,
			Error:    "update_failed",
			Phase:    "applying",
			BackupID: backupEntry.ID,
		})
		return err
	}
	
	// Step 3: Completed
	s.setFlowState(model.UpdateFlowState{
		State:    model.StateCompleted,
		Phase:    "completed",
		BackupID: backupEntry.ID,
	})
	
	return nil
}

// ApplyUpdate installs the pinned target version of Node-RED and runs a
// post-install vulnerability audit. If no resolved version is cached, the
// update is rejected so we never run an unpinned "latest" install.
func (s *UpdateService) ApplyUpdate() error {
	cached := s.GetCachedStatus()
	if cached.LatestVersion == "" {
		return fmt.Errorf("no resolved target version — run a version check first")
	}
	target := "node-red@" + cached.LatestVersion

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	_, err := s.runner.Run(ctx, "npm", "install", "-g", target)
	if err != nil {
		sanitized := s.sanitizeErrorMessage(err)
		return fmt.Errorf("%s", sanitized)
	}

	if err := s.postInstallAudit(ctx, cached.LatestVersion); err != nil {
		return err
	}

	return nil
}

// postInstallAudit audits the dependency tree of the freshly-installed Node-RED
// version and blocks the update only when npm reports a CONFIRMED critical
// vulnerability.
//
// npm cannot audit globally-installed packages (`npm audit -g` errors with
// EAUDITGLOBAL), so we resolve the pinned version's tree in an isolated
// workspace and audit that. Operational failures — npm missing, registry
// unreachable, tree unresolvable — are NOT vulnerabilities and must never block
// a legitimate update; only a parsed critical count > 0 blocks.
func (s *UpdateService) postInstallAudit(ctx context.Context, version string) error {
	auditDir, err := os.MkdirTemp("", "nrcc-audit-")
	if err != nil {
		return nil // cannot stage audit workspace → do not block
	}
	defer os.RemoveAll(auditDir)

	manifest := fmt.Sprintf(`{"name":"nrcc-audit","version":"0.0.0","private":true,"dependencies":{"node-red":"%s"}}`, version)
	if err := os.WriteFile(filepath.Join(auditDir, "package.json"), []byte(manifest), 0o600); err != nil {
		return nil // cannot write manifest → do not block
	}

	// Resolve the dependency tree without installing node_modules (lockfile only).
	if _, err := s.runner.Run(ctx, "npm", "install", "--package-lock-only", "--prefix", auditDir); err != nil {
		return nil // unresolvable tree (e.g. registry outage) → do not block
	}

	// `npm audit` exits non-zero when it finds vulnerabilities but still prints a
	// JSON report; we trust the parsed critical count, not the exit code.
	output, _ := s.runner.Run(ctx, "npm", "audit", "--audit-level=critical", "--omit=dev", "--json", "--prefix", auditDir)
	if critical := parseCriticalCount(output); critical > 0 {
		return fmt.Errorf("post-install audit found %d critical vulnerability(ies) — update blocked: %s", critical, truncateOutput(output, 200))
	}
	return nil
}

// parseCriticalCount extracts metadata.vulnerabilities.critical from an
// `npm audit --json` report. Any parse failure yields 0 so an unreadable or
// non-JSON audit output (e.g. an npm error string) never blocks an update.
func parseCriticalCount(output []byte) int {
	var report struct {
		Metadata struct {
			Vulnerabilities struct {
				Critical int `json:"critical"`
			} `json:"vulnerabilities"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(output, &report); err != nil {
		return 0
	}
	return report.Metadata.Vulnerabilities.Critical
}

func truncateOutput(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

// Helper functions

// sanitizeErrorMessage converts technical npm errors into user-friendly messages.
// This prevents raw exit codes and npm internals from leaking to the UI.
func (s *UpdateService) sanitizeErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	
	errStr := err.Error()
	
	// If it's a context deadline exceeded, the npm command timed out
	if strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "i/o timeout") {
		return "Update check timed out. Please try again."
	}
	
	// Exit status errors are npm-specific failures (234, 1, etc.) — hide the exit code
	if strings.Contains(errStr, "exit status") {
		return "Update check failed. Please ensure npm is properly installed and Node-RED is accessible."
	}
	
	// Connection/network errors
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "network") {
		return "Network error while checking for updates. Please check your internet connection."
	}
	
	// Default fallback for other errors
	return "Unable to check for updates at this time."
}

func (s *UpdateService) getInstalledVersionInternal(ctx context.Context) string {
	output, err := s.runner.Run(ctx, "npm", "list", "-g", "node-red", "--json")
	if err != nil {
		return "unknown"
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return "unknown"
	}

	if deps, ok := result["dependencies"].(map[string]interface{}); ok {
		if nodeRed, ok := deps["node-red"].(map[string]interface{}); ok {
			if version, ok := nodeRed["version"].(string); ok {
				return version
			}
		}
	}

	return "unknown"
}

func (s *UpdateService) getLatestVersionInternal(ctx context.Context) (string, error) {
	output, err := s.runner.Run(ctx, "npm", "view", "node-red", "version")
	if err != nil {
		return "", err
	}

	version := strings.TrimSpace(string(output))
	version = strings.Trim(version, "\n\"")

	return version, nil
}

// compareVersions compares two semver versions
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func (s *UpdateService) compareVersions(v1, v2 string) int {
	v1Parts := strings.Split(strings.TrimPrefix(v1, "v"), ".")
	v2Parts := strings.Split(strings.TrimPrefix(v2, "v"), ".")

	// Pad with zeros
	for len(v1Parts) < len(v2Parts) {
		v1Parts = append(v1Parts, "0")
	}
	for len(v2Parts) < len(v1Parts) {
		v2Parts = append(v2Parts, "0")
	}

	for i := 0; i < len(v1Parts); i++ {
		var v1Num, v2Num int
		fmt.Sscanf(v1Parts[i], "%d", &v1Num)
		fmt.Sscanf(v2Parts[i], "%d", &v2Num)

		if v1Num < v2Num {
			return -1
		} else if v1Num > v2Num {
			return 1
		}
	}

	return 0
}
