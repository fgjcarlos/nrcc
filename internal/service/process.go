package service

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/ui"
)

// ProcessManager manages the Node-RED child process lifecycle.
//
// Design invariants:
//   - pm.currentCmd is the live *exec.Cmd; nil when stopped.
//   - waitForExit is the sole caller of cmd.Wait() for each cmd instance.
//   - Stop() signals intent via stopCh (close), then waits on doneCh.
//   - stopCh and doneCh are created fresh on every Start(), avoiding the
//     "closed channel read always succeeds" pitfall.
type ProcessManager struct {
	// immutable after construction
	nodeRedCmd   string
	dataDir      string
	logBuffer    *LogBuffer
	maxRestarts  int
	restartDelay time.Duration

	// set via setter, guarded by mu
	envSvc *EnvService

	// process state — guarded by mu
	mu           sync.Mutex
	currentCmd   *exec.Cmd
	running      bool
	externalMode bool // true when attaching to a pre-existing Node-RED process
	startedAt    time.Time
	restartCount int
	lastError    error
	stopCh       chan struct{} // closed to signal intentional stop to waitForExit
	doneCh       chan struct{} // closed by waitForExit when the process fully exits

	// version fetched once
	version     string
	versionOnce sync.Once
}

// NewProcessManager creates a ProcessManager. cmd is the node-red executable
// path; if empty "node-red" is used.
func NewProcessManager(cmd, dataDir string, logBuffer *LogBuffer) *ProcessManager {
	if cmd == "" {
		cmd = "node-red"
	}
	return &ProcessManager{
		nodeRedCmd:   cmd,
		dataDir:      dataDir,
		logBuffer:    logBuffer,
		maxRestarts:  10,
		restartDelay: 2 * time.Second,
		doneCh:       closedChan(), // sentinel so Stop() doesn't block when nothing is running
	}
}

// closedChan returns an already-closed channel used as a sentinel.
func closedChan() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// isPortInUse checks if a TCP port is already bound on localhost.
func isPortInUse(port string) bool {
	conn, err := net.DialTimeout("tcp", "localhost:"+port, time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// SetEnvService wires an EnvService so stored variables are injected into
// the node-red process environment on every Start().
func (pm *ProcessManager) SetEnvService(svc *EnvService) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.envSvc = svc
}

// IsExternalMode returns true if ProcessManager is attached to an externally managed Node-RED.
func (pm *ProcessManager) IsExternalMode() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.externalMode
}

// Start starts the Node-RED process. Returns an error if already running.
func (pm *ProcessManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.startLocked(true)
}

// startLocked starts the process. Must be called with pm.mu held.
// resetCounter resets the crash counter (true for user-initiated starts,
// false for auto-restarts so the backoff counter is preserved).
func (pm *ProcessManager) startLocked(resetCounter bool) error {
	if pm.running {
		pid := pm.currentCmd.Process.Pid
		return fmt.Errorf("process already running (PID: %d)", pid)
	}

	// Check if Node-RED is already running externally
	nodeRedPort := "1880"
	if isPortInUse(nodeRedPort) {
		// Attach to existing instance
		pm.externalMode = true
		pm.running = true
		// Try to find the PID for status reporting
		running, pid := processRunning("node-red")
		if running {
			ui.Info(fmt.Sprintf("Node-RED already running (PID: %d) — attaching", pid))
		} else {
			ui.Info("Node-RED already running on :1880 — attaching")
		}
		// Push a synthetic log entry to the buffer so the UI shows context
		pm.logBuffer.Push(model.LogEntry{
			ID:        "nrcc-attach-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     "info",
			Source:    "nrcc",
			Message:   "Attached to existing Node-RED instance on :1880",
		})
		return nil
	}

	// Normal path: start as child process
	pm.externalMode = false

	settingsPath := filepath.Join(pm.dataDir, "settings.js")
	if err := ensureSettings(settingsPath); err != nil {
		return fmt.Errorf("failed to create settings.js: %w", err)
	}

	cmd := exec.Command(pm.nodeRedCmd,
		"--userDir", pm.dataDir,
		"--settings", settingsPath,
		"--port", "1880",
	)

	// Build environment with proper precedence:
	// Layer 1: OS environ (base)
	// Layer 2: config.json vars (stored in configSvc)
	// Layer 3: .env vars (highest priority)
	envMap := make(map[string]string)

	// Layer 1: OS environ
	for _, pair := range os.Environ() {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Layer 2: config.json vars
	if pm.envSvc != nil {
		if stored, err := pm.envSvc.GetAll(); err == nil {
			for k, v := range stored {
				envMap[k] = v
			}
		}
	}

	// Layer 3: .env vars (highest priority — TAREA 2b)
	dotenvPath := filepath.Join(pm.dataDir, ".env")
	if dotenvVars, err := parseEnvFile(dotenvPath); err == nil {
		for k, v := range dotenvVars {
			envMap[k] = v
		}
	}

	// Convert map to env slice
	cmd.Env = make([]string, 0, len(envMap))
	for k, v := range envMap {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start node-red: %w", err)
	}

	// Fresh channels for this process lifetime.
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})

	pm.currentCmd = cmd
	pm.stopCh = stopCh
	pm.doneCh = doneCh
	pm.running = true
	pm.startedAt = time.Now()
	pm.lastError = nil
	if resetCounter {
		pm.restartCount = 0
	}

	go pm.consumeLogs(stdoutPipe, "stdout")
	go pm.consumeLogs(stderrPipe, "stderr")
	// Pass local copies of the channels — waitForExit must not read pm.stopCh/doneCh
	// after the lock is released, as a new Start() could replace them.
	go pm.waitForExit(cmd, stopCh, doneCh)

	return nil
}

// Stop gracefully stops the node-red process and waits for it to exit.
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	if !pm.running || pm.currentCmd == nil {
		pm.mu.Unlock()
		return nil
	}

	// If in external mode, just detach without killing the process
	if pm.externalMode {
		pm.running = false
		pm.externalMode = false
		pm.mu.Unlock()
		ui.Info("Detached from external Node-RED instance (process not stopped)")
		return nil
	}

	stopCh := pm.stopCh
	doneCh := pm.doneCh
	proc := pm.currentCmd.Process
	pm.mu.Unlock()

	// Close stopCh BEFORE sending the signal so waitForExit sees it as intentional.
	close(stopCh)

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		_ = proc.Kill()
	}

	// Wait for waitForExit to call cmd.Wait() and close doneCh.
	select {
	case <-doneCh:
		return nil
	case <-time.After(5 * time.Second):
		_ = proc.Kill()
		<-doneCh // still wait so cmd.Wait() finishes before we return
		return nil
	}
}

// Restart stops and starts the node-red process.
func (pm *ProcessManager) Restart() error {
	pm.mu.Lock()
	if pm.externalMode {
		pm.mu.Unlock()
		return fmt.Errorf("cannot restart an externally managed Node-RED instance")
	}
	pm.mu.Unlock()

	if err := pm.Stop(); err != nil {
		return fmt.Errorf("stop failed: %w", err)
	}
	return pm.Start()
}

// Status returns the current runtime status.
func (pm *ProcessManager) Status() model.RuntimeStatus {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	status := "stopped"
	var pid int
	var uptime int64
	var startedAt string

	if pm.running && pm.currentCmd != nil {
		status = "running"
		pid = pm.currentCmd.Process.Pid
		uptime = int64(time.Since(pm.startedAt).Seconds())
		startedAt = pm.startedAt.UTC().Format(time.RFC3339)
	} else if pm.externalMode {
		status = "running (external)"
		// Try to get PID of external process
		running, extPid := processRunning("node-red")
		if running {
			pid = extPid
		}
	}

	return model.RuntimeStatus{
		Status:    status,
		PID:       pid,
		Uptime:    uptime,
		Version:   pm.version,
		StartedAt: startedAt,
	}
}

// Version returns the node-red version string (fetched once and cached).
func (pm *ProcessManager) Version() string {
	pm.versionOnce.Do(func() {
		out, err := exec.Command(pm.nodeRedCmd, "--version").Output()
		if err == nil {
			pm.version = strings.TrimSpace(string(out))
		}
	})
	return pm.version
}

// consumeLogs reads lines from a pipe and pushes them to the log buffer.
func (pm *ProcessManager) consumeLogs(r io.Reader, stream string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		pm.logBuffer.Push(model.LogEntry{
			ID:        fmt.Sprintf("%s-%d", stream, time.Now().UnixNano()),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     parseLogLevel(line),
			Source:    stream,
			Message:   line,
		})
	}
}

// parseLogLevel extracts the log level from a node-red output line.
func parseLogLevel(line string) string {
	switch {
	case strings.Contains(line, "[error]"):
		return "error"
	case strings.Contains(line, "[warn]"):
		return "warn"
	case strings.Contains(line, "[debug]"):
		return "debug"
	default:
		return "info"
	}
}

// waitForExit is the sole goroutine that calls cmd.Wait(). It handles
// auto-restart with exponential backoff on unexpected exits.
//
// stopCh and doneCh are passed by value (captured at start time) so this
// goroutine is not affected by a subsequent Start() creating new channels.
func (pm *ProcessManager) waitForExit(cmd *exec.Cmd, stopCh, doneCh chan struct{}) {
	err := cmd.Wait()

	pm.mu.Lock()
	pm.running = false
	pm.currentCmd = nil
	pm.lastError = err
	restartCount := pm.restartCount
	pm.mu.Unlock()

	close(doneCh) // unblocks Stop() if it is waiting

	// Was this an intentional stop?
	select {
	case <-stopCh:
		return
	default:
	}

	// Unexpected crash — attempt auto-restart with exponential backoff.
	if restartCount >= pm.maxRestarts {
		pm.logBuffer.Push(model.LogEntry{
			ID:        fmt.Sprintf("nrcc-%d", time.Now().UnixNano()),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     "error",
			Source:    "nrcc",
			Message:   fmt.Sprintf("node-red crashed %d times — giving up automatic restarts", restartCount),
		})
		return
	}

	// Backoff: 2s, 4s, 8s, 16s … capped at 60s
	delay := pm.restartDelay * (1 << uint(restartCount))
	if delay > 60*time.Second {
		delay = 60 * time.Second
	}

	pm.logBuffer.Push(model.LogEntry{
		ID:        fmt.Sprintf("nrcc-%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     "warn",
		Source:    "nrcc",
		Message:   fmt.Sprintf("node-red exited unexpectedly (attempt %d/%d) — restarting in %s", restartCount+1, pm.maxRestarts, delay),
	})

	time.Sleep(delay)

	pm.mu.Lock()
	pm.restartCount++
	if startErr := pm.startLocked(false); startErr != nil {
		pm.lastError = startErr
	}
	pm.mu.Unlock()
}

// GetLogs returns the last `limit` log lines as []string.
// If limit <= 0, returns all available logs.
// If logBuffer is nil or empty, returns empty slice.
func (pm *ProcessManager) GetLogs(limit int) []string {
	if pm.logBuffer == nil {
		return []string{}
	}

	var logEntries []model.LogEntry
	if limit <= 0 {
		logEntries = pm.logBuffer.All()
	} else {
		logEntries = pm.logBuffer.Recent(limit)
	}

	result := make([]string, len(logEntries))
	for i, entry := range logEntries {
		result[i] = entry.Message
	}
	return result
}

// ensureSettings creates a minimal settings.js if one doesn't already exist.
// node-red requires this file when launched with --settings.
func ensureSettings(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	const defaultSettings = `module.exports = {
    // Node-RED settings — managed by nrcc
    // See https://nodered.org/docs/user-guide/runtime/configuration
    flowFile: 'flows.json',
    credentialSecret: false,
    flowFilePretty: true,
    adminAuth: null,
    editorTheme: {
        projects: { enabled: false }
    },
    logging: {
        console: { level: 'info', metrics: false, audit: false }
    },
    exportGlobalContextKeys: false,
    externalModules: {}
}
`
	return os.WriteFile(path, []byte(defaultSettings), 0644)
}
