package service

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/ui"
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

	// restart history (ring buffer, capacity 50)
	tracker *restartTracker

	// cumulativeRestarts is the durable lifetime count of auto-restarts.
	// It is SEPARATE from restartCount (the backoff/give-up counter).
	// Persisted via restartCountStore; never reset by the user-start path.
	cumulativeRestarts int
	restartStore       *restartCountStore

	// version fetched once
	version     string
	versionOnce sync.Once
}

// NewProcessManager creates a ProcessManager. cmd is the node-red executable
// path; if empty "node-red" is used.
func NewProcessManager(cmd, dataDir string) *ProcessManager {
	if cmd == "" {
		cmd = "node-red"
	}
	store := newRestartCountStore(dataDir)
	return &ProcessManager{
		nodeRedCmd:         cmd,
		dataDir:            dataDir,
		maxRestarts:        10,
		restartDelay:       2 * time.Second,
		doneCh:             closedChan(), // sentinel so Stop() doesn't block when nothing is running
		// Two sources of truth about restarts, on purpose (see restart_tracker.go):
		//   - tracker: in-memory ring buffer of recent events for the
		//     runtime history endpoint; bounded, ephemeral.
		//   - restartStore: persistent monotonic counter that survives
		//     process restarts; exposed as CumulativeRestarts().
		tracker:            newRestartTracker(50),
		restartStore:       store,
		cumulativeRestarts: store.Load(),
	}
}

// CumulativeRestarts returns the total number of auto-restarts over the
// lifetime of this installation. This is the DURABLE counter (separate from
// the backoff counter pm.restartCount which resets on user-initiated starts).
func (pm *ProcessManager) CumulativeRestarts() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.cumulativeRestarts
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
	_ = conn.Close()
	return true
}

// nodeRedRuntime holds the explicit Node-RED runtime contract honored by
// ProcessManager (issue #429). One runtime per container/instance:
//   - Port:       NODE_RED_PORT  (default 1880)
//   - UserDir:    NODE_RED_USER_DIR (default dataDir)
//   - Settings:   NODE_RED_SETTINGS (default <userDir>/settings.js)
type nodeRedRuntime struct {
	Port         string
	UserDir      string
	SettingsPath string
}

// resolveNodeRedRuntime builds the Node-RED runtime contract from the
// env precedence layers ProcessManager.Start already uses (OS env +
// stored .env). dataDir is the fallback userDir and settings parent.
//
// Validation:
//   - Port: integer 1-65535.
//   - UserDir: non-empty (resolved to absolute relative to cwd if needed).
//   - Settings: must resolve inside UserDir unless an absolute override
//     is provided.
func resolveNodeRedRuntime(env map[string]string, dataDir string) (nodeRedRuntime, error) {
	dataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nodeRedRuntime{}, fmt.Errorf("resolve dataDir: %w", err)
	}

	rt := nodeRedRuntime{
		Port:    "1880",
		UserDir: dataDir,
	}

	if v := strings.TrimSpace(env["NODE_RED_PORT"]); v != "" {
		n, perr := strconv.Atoi(v)
		if perr != nil || n < 1 || n > 65535 {
			return nodeRedRuntime{}, fmt.Errorf("invalid NODE_RED_PORT %q: must be integer 1-65535", v)
		}
		rt.Port = v
	}

	if v := strings.TrimSpace(env["NODE_RED_USER_DIR"]); v != "" {
		abs := v
		if !filepath.IsAbs(abs) {
			abs, err = filepath.Abs(abs)
			if err != nil {
				return nodeRedRuntime{}, fmt.Errorf("resolve NODE_RED_USER_DIR: %w", err)
			}
		}
		rt.UserDir = abs
	}

	settings := filepath.Join(rt.UserDir, "settings.js")
	if v := strings.TrimSpace(env["NODE_RED_SETTINGS"]); v != "" {
		settings = v
		if !filepath.IsAbs(settings) {
			settings = filepath.Join(rt.UserDir, settings)
		}
	}
	rt.SettingsPath = settings

	return rt, nil
}

// SetEnvService wires an EnvService so stored variables are injected into
// the node-red process environment on every Start().
func (pm *ProcessManager) SetEnvService(svc *EnvService) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.envSvc = svc
}

// envForChild builds the environment map that will be passed to the
// Node-RED child process, with the precedence:
//   1. OS environ (base)
//   2. config.json vars (EnvService)
//   3. .env file in dataDir (highest)
func (pm *ProcessManager) envForChild() map[string]string {
	envMap := make(map[string]string)
	for _, pair := range os.Environ() {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	if pm.envSvc != nil {
		if stored, err := pm.envSvc.GetAll(); err == nil {
			for k, v := range stored {
				envMap[k] = v
			}
		}
	}
	dotenvPath := filepath.Join(pm.dataDir, ".env")
	if dotenvVars, err := parseEnvFile(dotenvPath); err == nil {
		for k, v := range dotenvVars {
			envMap[k] = v
		}
	}
	return envMap
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

	// Resolve explicit Node-RED runtime contract (NODE_RED_PORT/USER_DIR/SETTINGS).
	// See resolveNodeRedRuntime for precedence and validation.
	rt, err := resolveNodeRedRuntime(pm.envForChild(), pm.dataDir)
	if err != nil {
		return fmt.Errorf("invalid Node-RED runtime: %w", err)
	}

	// Check if Node-RED is already running externally on the configured port
	if isPortInUse(rt.Port) {
		// Attach to existing instance
		pm.externalMode = true
		pm.running = true
		// Try to find the PID for status reporting
		running, pid := processRunning("node-red")
		if running {
			ui.Info(fmt.Sprintf("Node-RED already running (PID: %d) — attaching", pid))
		} else {
			ui.Info(fmt.Sprintf("Node-RED already running on :%s — attaching", rt.Port))
		}
		return nil
	}

	// Normal path: start as child process
	pm.externalMode = false

	if err := ensureSettings(rt.SettingsPath); err != nil {
		return fmt.Errorf("failed to create settings.js: %w", err)
	}

	cmd := exec.Command(pm.nodeRedCmd,
		"--userDir", rt.UserDir,
		"--settings", rt.SettingsPath,
		"--port", rt.Port,
	)

	// Build environment with proper precedence:
	// Layer 1: OS environ (base)
	// Layer 2: config.json vars (stored in configSvc)
	// Layer 3: .env vars (highest priority)
	envMap := pm.envForChild()

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

	// Drain the child stdout/stderr into io.Discard so the kernel pipe buffer
	// never fills and `cmd.Wait()` exits cleanly. The captured lines are no
	// longer exposed via a /logs page; Node-RED's own stdout is still visible
	// in `docker logs nrcc`.
	drainDone := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(io.Discard, stdoutPipe); drainDone <- struct{}{} }()
	go func() { _, _ = io.Copy(io.Discard, stderrPipe); drainDone <- struct{}{} }()
	// Pass local copies of the channels — waitForExit must not read pm.stopCh/doneCh
	// after the lock is released, as a new Start() could replace them.
	go pm.waitForExit(cmd, stopCh, doneCh)
	// Best-effort wait for the drainers to finish when the child exits, so
	// their goroutines don't leak past the lifetime of the cmd.
	go func() {
		<-doneCh
		<-drainDone
		<-drainDone
	}()

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
		Status:              status,
		PID:                 pid,
		Uptime:              uptime,
		RestartCount:        pm.cumulativeRestarts,
		ConsecutiveFailures: pm.restartCount,
		Version:             pm.version,
		StartedAt:           startedAt,
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

	// Record the unexpected exit in the restart history.
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	pm.tracker.push(model.RestartEvent{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		ExitCode:    exitCode,
		Attempt:     restartCount + 1,
		MaxAttempts: pm.maxRestarts,
	})

	// Unexpected crash — attempt auto-restart with exponential backoff.
	if restartCount >= pm.maxRestarts {
		ui.Errorf("node-red crashed %d times — giving up automatic restarts", restartCount)
		return
	}

	// Backoff: 2s, 4s, 8s, 16s … capped at 60s
	delay := pm.restartDelay * (1 << uint(restartCount))
	if delay > 60*time.Second {
		delay = 60 * time.Second
	}

	ui.Warnf("node-red exited unexpectedly (attempt %d/%d) — restarting in %s", restartCount+1, pm.maxRestarts, delay)

	time.Sleep(delay)

	pm.mu.Lock()
	pm.restartCount++
	pm.cumulativeRestarts++
	if saveErr := pm.restartStore.Save(pm.cumulativeRestarts); saveErr != nil {
		// Best-effort — a save failure is logged but never fatal.
		ui.Warnf("restart count store: save failed: %v", saveErr)
	}
	if startErr := pm.startLocked(false); startErr != nil {
		pm.lastError = startErr
	}
	pm.mu.Unlock()
}

// RestartEvents returns the recorded unexpected-exit events in chronological
// order (oldest first), up to the tracker's capacity of 50.
func (pm *ProcessManager) RestartEvents() []model.RestartEvent {
	return pm.tracker.restartEvents()
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
