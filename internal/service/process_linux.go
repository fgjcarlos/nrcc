//go:build linux

package service

import (
	"bufio"
	"context"
	"errors"
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

// ErrForeignListener is returned by ProcessManager.Restart when the runtime
// port is already bound by a process this manager does not own (issue #918).
// Attaching to a foreign listener would let an unrelated process satisfy
// readiness, so the strict restart path refuses instead.
var ErrForeignListener = errors.New("node-red port is already bound by a foreign process")

// Env precedence for the Node-RED child process (and any helper that
// resolves the same env layers, e.g. EnvService.activeNodeRedUserDir):
//   1. OS environ (base)
//   2. EnvService.GetAll() — variables managed by NRCC
//   3. dataDir/.env file (highest priority)
// Single source of truth: mergeEnvLayers below.

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
	// stableUptime is how long a child must run before its exit no longer
	// counts against the crash budget (issue #1013).
	stableUptime time.Duration

	// output receives the child's stdout and stderr, prefixed per line, so
	// Node-RED start-up errors reach `docker logs nrcc` (issue #1015).
	output io.Writer

	// set via setter, guarded by mu
	envSvc *EnvService

	// process state — guarded by mu
	mu           sync.Mutex
	currentCmd   *exec.Cmd
	running      bool
	externalMode bool // true when attaching to a pre-existing Node-RED process
	runtimePort  int  // port resolved for the most recent successful Start; guarded by mu
	startedAt    time.Time
	restartCount int
	// restartsExhausted is set when automatic restarts give up and cleared
	// by the next successful user-initiated Start; /healthz reports it.
	restartsExhausted bool
	lastError         error
	stopCh            chan struct{} // closed to signal intentional stop to waitForExit
	stopOnce          *sync.Once    // guards closing stopCh so Stop is idempotent
	doneCh            chan struct{} // closed by waitForExit when the process fully exits

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
		nodeRedCmd:   cmd,
		dataDir:      dataDir,
		maxRestarts:  10,
		restartDelay: 2 * time.Second,
		output:       os.Stdout,
		stableUptime: 5 * time.Minute,
		doneCh:       closedChan(), // sentinel so Stop() doesn't block when nothing is running
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

// EffectiveRuntimePort returns the port resolved for the most recent
// successful ProcessManager.Start. It is the value passed to Node-RED's
// --port option after OS, stored-variable, and .env precedence is applied.
// The last launch value is retained so readiness checks do not re-read a
// changed .env file or inspect process metadata. Zero means Start has not
// succeeded yet.
func (pm *ProcessManager) EffectiveRuntimePort() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.runtimePort
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
//  1. OS environ (base)
//  2. config.json vars (EnvService)
//  3. .env file in dataDir (highest)
func (pm *ProcessManager) envForChild() map[string]string {
	var stored map[string]string
	if pm.envSvc != nil {
		stored, _ = pm.envSvc.GetAll()
	}
	dotenvVars, _ := parseEnvFile(filepath.Join(pm.dataDir, ".env"))
	return mergeEnvLayers(os.Environ(), stored, dotenvVars)
}

func mergeEnvLayers(osEnviron []string, stored map[string]string, dotenvVars map[string]string) map[string]string {
	envMap := make(map[string]string, len(osEnviron)+len(stored)+len(dotenvVars))
	for _, pair := range osEnviron {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	for k, v := range stored {
		envMap[k] = v
	}
	for k, v := range dotenvVars {
		envMap[k] = v
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
// Start may attach to an already-running external Node-RED instance bound
// to the configured port (attach is the documented UX for manual starts).
func (pm *ProcessManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if err := pm.startLocked(true, true); err != nil {
		return err
	}
	pm.restartsExhausted = false
	return nil
}

// RestartBudgetExhausted reports whether automatic restarts gave up after
// repeated crashes and no user-initiated start has succeeded since.
func (pm *ProcessManager) RestartBudgetExhausted() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.restartsExhausted
}

// ranStableLocked reports whether the current child has been up for the
// stable-uptime threshold. Must be called with pm.mu held.
func (pm *ProcessManager) ranStableLocked(now time.Time) bool {
	return !pm.startedAt.IsZero() && now.Sub(pm.startedAt) >= pm.stableUptime
}

// startLocked starts the process. Must be called with pm.mu held.
// resetCounter resets the crash counter (true for user-initiated starts,
// false for auto-restarts so the backoff counter is preserved).
// attachAllowed controls the occupied-port policy: user-facing Start()
// attaches to an existing listener, while the strict restart path
// (issue #918) refuses so readiness can never be satisfied by a
// foreign process.
func (pm *ProcessManager) startLocked(resetCounter bool, attachAllowed bool) error {
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
	runtimePort, err := strconv.Atoi(rt.Port)
	if err != nil {
		return fmt.Errorf("invalid resolved Node-RED runtime port %q: %w", rt.Port, err)
	}

	// Check if Node-RED is already running externally on the configured port
	if isPortInUse(rt.Port) {
		if !attachAllowed {
			// Strict path (restart): we just stopped the child we own, so any
			// remaining listener is a foreign process. Attaching would let it
			// satisfy the readiness probe; refuse instead (issue #918).
			return fmt.Errorf("%w: port %s", ErrForeignListener, rt.Port)
		}
		// Attach to existing instance
		pm.externalMode = true
		pm.running = true
		pm.runtimePort = runtimePort
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

	// #nosec G204,G702 -- pm.nodeRedCmd is operator-supplied via configuration; rt fields are derived from operator-controlled dataDir.
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

	// Wait returns only after exec has copied all output into these pipes,
	// unlike StdoutPipe, whose read end Wait closes while still unread.
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW
	cmd.WaitDelay = outputWaitDelay

	if err := cmd.Start(); err != nil {
		_ = stdoutW.Close()
		_ = stderrW.Close()
		return fmt.Errorf("failed to start node-red: %w", err)
	}

	// Fresh channels for this process lifetime.
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})

	pm.currentCmd = cmd
	pm.runtimePort = runtimePort
	pm.stopCh = stopCh
	pm.stopOnce = &sync.Once{}
	pm.doneCh = doneCh
	pm.running = true
	pm.startedAt = time.Now()
	pm.lastError = nil
	if resetCounter {
		pm.restartCount = 0
	}

	// Forward the child stdout/stderr to the NRCC output (the container log;
	// Docker rotates it) so a failed start explains itself. The forwarders end
	// at EOF when waitForExit closes the pipe writers.
	go forwardOutput(pm.output, stdoutR, "node-red: ")
	go forwardOutput(pm.output, stderrR, "node-red: ")
	// Pass local copies of the channels — waitForExit must not read pm.stopCh/doneCh
	// after the lock is released, as a new Start() could replace them.
	go pm.waitForExit(cmd, stopCh, doneCh)

	return nil
}

// outputWaitDelay bounds how long Wait copies output after the child exits:
// a grandchild holding the fd must not block exit detection and restart.
const outputWaitDelay = 2 * time.Second

// forwardOutput copies src to dst until EOF, starting each line with prefix.
// Each line goes out in one Write so the stdout and stderr forwarders do not
// interleave mid-line; lines longer than the read buffer are split into
// chunks. Write errors are ignored: src must keep draining or the child
// blocks on a full pipe.
func forwardOutput(dst io.Writer, src io.Reader, prefix string) {
	r := bufio.NewReader(src)
	lineStart := true
	for {
		chunk, err := r.ReadSlice('\n')
		if len(chunk) > 0 {
			buf := chunk
			if lineStart {
				buf = append([]byte(prefix), chunk...)
			}
			_, _ = dst.Write(buf)
			lineStart = chunk[len(chunk)-1] == '\n'
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		if err != nil {
			if !lineStart {
				_, _ = dst.Write([]byte("\n"))
			}
			return
		}
	}
}

// Stop gracefully stops the node-red process and waits for it to exit,
// bounded by ctx (issue #918). When the child ignores SIGTERM and outlives
// ctx, Stop returns the context error instead of blocking forever; the
// child is killed best-effort before returning.
func (pm *ProcessManager) Stop(ctx context.Context) error {
	pm.mu.Lock()
	if !pm.running || pm.currentCmd == nil {
		// No child, but a crashed one may be waiting out its restart
		// backoff: closing its stopCh cancels that restart (issue #1014).
		if pm.stopOnce != nil {
			stopCh := pm.stopCh
			pm.stopOnce.Do(func() { close(stopCh) })
		}
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
	stopOnce := pm.stopOnce
	doneCh := pm.doneCh
	proc := pm.currentCmd.Process
	pm.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	// Close stopCh BEFORE sending the signal so waitForExit sees it as
	// intentional. The once makes Stop idempotent across retries (a caller
	// may re-invoke Stop after a deadline expiry).
	stopOnce.Do(func() { close(stopCh) })

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		_ = proc.Kill()
	}

	// Wait for waitForExit to call cmd.Wait() and close doneCh. The wait is
	// bounded by ctx so a hung child cannot outlive the recovery deadline.
	select {
	case <-doneCh:
		return nil
	case <-time.After(stopGraceTimeout):
		_ = proc.Kill()
		select {
		case <-doneCh:
			return nil
		case <-ctx.Done():
			// The child ignored SIGKILL escalation within the caller's
			// deadline. Report the context error honestly; cmd.Wait() still
			// runs in waitForExit, so no goroutine owns the wait twice.
			return fmt.Errorf("stop node-red: %w", ctx.Err())
		}
	case <-ctx.Done():
		_ = proc.Kill()
		return fmt.Errorf("stop node-red: %w", ctx.Err())
	}
}

// stopGraceTimeout is the SIGTERM→SIGKILL grace window Stop waits before
// escalating, independent of the caller's context deadline.
const stopGraceTimeout = 5 * time.Second

// Restart stops and starts the node-red process, bounded by ctx
// (issue #918). The start leg uses the strict occupied-port policy so a
// foreign listener on the runtime port fails the restart instead of being
// adopted.
func (pm *ProcessManager) Restart(ctx context.Context) error {
	pm.mu.Lock()
	if pm.externalMode {
		pm.mu.Unlock()
		return fmt.Errorf("cannot restart an externally managed Node-RED instance")
	}
	pm.mu.Unlock()

	if err := pm.Stop(ctx); err != nil {
		return fmt.Errorf("stop failed: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("restart: %w", err)
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if err := pm.startLocked(true, false); err != nil {
		return err
	}
	pm.restartsExhausted = false
	return nil
}

// Status returns the current runtime status.
func (pm *ProcessManager) Status() model.RuntimeStatus {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	status := "stopped"
	var pid int
	var uptime int64
	var startedAt string
	failures := pm.restartCount

	if pm.running && pm.currentCmd != nil {
		if pm.ranStableLocked(time.Now()) {
			failures = 0 // the next exit resets the budget (issue #1013)
		}
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
		ConsecutiveFailures: failures,
		Version:             pm.version,
		StartedAt:           startedAt,
	}
}

// Version returns the node-red version string (fetched once and cached).
func (pm *ProcessManager) Version() string {
	pm.versionOnce.Do(func() {
		managed := pm.ManagedRuntime()
		// #nosec G204 -- managed.Executable is derived from the operator-supplied command; "--version" is a fixed flag.
		out, err := exec.Command(managed.Executable, "--version").Output()
		if err == nil {
			pm.version = strings.TrimSpace(string(out))
		}
	})
	return pm.version
}

// ManagedRuntime returns the installation authority for the Node-RED process
// this manager launches.
func (pm *ProcessManager) ManagedRuntime() ManagedRuntime {
	return ResolveManagedRuntime(pm.nodeRedCmd, pm.dataDir)
}

// waitForExit is the sole goroutine that calls cmd.Wait(). It handles
// auto-restart with exponential backoff on unexpected exits.
//
// stopCh and doneCh are passed by value (captured at start time) so this
// goroutine is not affected by a subsequent Start() creating new channels.
func (pm *ProcessManager) waitForExit(cmd *exec.Cmd, stopCh, doneCh chan struct{}) {
	err := cmd.Wait()
	// EOF lets the forwarders flush a trailing line without newline and exit.
	for _, w := range []io.Writer{cmd.Stdout, cmd.Stderr} {
		if c, ok := w.(*io.PipeWriter); ok {
			_ = c.Close()
		}
	}

	pm.mu.Lock()
	// The budget counts crashes close together, not over the lifetime of
	// the container: a child that ran stably starts a fresh budget (#1013).
	if pm.ranStableLocked(time.Now()) {
		pm.restartCount = 0
	}
	pm.running = false
	pm.currentCmd = nil
	pm.lastError = err
	restartCount := pm.restartCount
	// Set in the same critical section as running=false so a concurrent
	// user Start cannot clear the flag before it is raised.
	select {
	case <-stopCh:
	default:
		if restartCount >= pm.maxRestarts {
			pm.restartsExhausted = true
		}
	}
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

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-stopCh:
		return // Stop cancelled the pending restart (issue #1014)
	case <-timer.C:
	}

	pm.mu.Lock()
	// Re-check under the lock: Stop may have run after the timer fired, or
	// a user Start may already own a newer child or an attached runtime.
	if pm.running || pm.stopCh != stopCh || isClosed(stopCh) {
		pm.mu.Unlock()
		return
	}
	pm.restartCount++
	pm.cumulativeRestarts++
	if saveErr := pm.restartStore.Save(pm.cumulativeRestarts); saveErr != nil {
		// Best-effort — a save failure is logged but never fatal.
		ui.Warnf("restart count store: save failed: %v", saveErr)
	}
	if startErr := pm.startLocked(false, true); startErr != nil {
		// No watcher runs after a failed launch: Node-RED is down for good.
		pm.lastError = startErr
		pm.restartsExhausted = true
		ui.Errorf("node-red auto-restart failed — giving up automatic restarts: %v", startErr)
	}
	pm.mu.Unlock()
}

func isClosed(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
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
	return os.WriteFile(path, []byte(defaultSettings), 0600)
}
