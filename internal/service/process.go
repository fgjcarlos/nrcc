package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

type ProcessConfig struct {
	DataDir string
	Port    int
}

type ProcessManager struct {
	cfg        ProcessConfig
	httpClient *http.Client
	logService *LogService

	mu         sync.RWMutex
	cmd        *exec.Cmd
	startedAt  time.Time
	lastError  string
	lastExit   string
	version    string
	binaryPath string
	logs       *ringBuffer
}

func NewProcessManager(cfg ProcessConfig) *ProcessManager {
	return &ProcessManager{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		logs: newRingBuffer(500),
	}
}

// SetLogService injects the LogService for structured logging (nil-safe)
func (pm *ProcessManager) SetLogService(ls *LogService) {
	pm.logService = ls
}

func (pm *ProcessManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.cmd != nil && pm.cmd.Process != nil {
		if pm.cmd.ProcessState == nil || !pm.cmd.ProcessState.Exited() {
			return fmt.Errorf("node-red runtime is already running")
		}
	}

	scriptPath := filepath.Join(pm.cfg.DataDir, "node_modules", "node-red", "red.js")
	if !platform.Exists(scriptPath) {
		return fmt.Errorf("node-red runtime not found at %s", scriptPath)
	}

	nodePath, err := exec.LookPath("node")
	if err != nil {
		return fmt.Errorf("resolve node executable: %w", err)
	}

	cmd := exec.Command(
		nodePath,
		scriptPath,
		"-u", pm.cfg.DataDir,
		"-p", fmt.Sprintf("%d", pm.cfg.Port),
	)
	cmd.Dir = pm.cfg.DataDir
	cmd.Env = pm.buildEnv()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("attach stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("attach stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start node-red: %w", err)
	}

	pm.cmd = cmd
	pm.startedAt = time.Now().UTC()
	pm.lastError = ""
	pm.lastExit = ""
	pm.binaryPath = scriptPath
	pm.logs.Add("runtime started")

	// Emit lifecycle event
	if pm.logService != nil {
		entry := model.LogEntry{
			Level:     model.LogLevelInfo,
			Source:    model.SourceRuntime,
			Event:     model.EventRuntimeLifecycle,
			Message:   "Node-RED process started",
			Timestamp: time.Now().UTC(),
		}
		_ = pm.logService.Write(entry)
	}

	go pm.captureOutput(stdout, "stdout")
	go pm.captureOutput(stderr, "stderr")
	go pm.waitForExit(cmd)

	version, _ := pm.readInstalledVersion()
	pm.version = version

	return nil
}

func (pm *ProcessManager) Stop() error {
	pm.mu.RLock()
	cmd := pm.cmd
	pm.mu.RUnlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return nil
	}

	pm.logs.Add("runtime stopping")
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("stop node-red: %w", err)
	}

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if !pm.Status().Running {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}

	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("kill node-red after timeout: %w", err)
	}
	return nil
}

func (pm *ProcessManager) Restart() error {
	pm.logs.Add("runtime restart requested")
	if err := pm.Stop(); err != nil {
		return err
	}
	return pm.Start()
}

func (pm *ProcessManager) Shutdown(ctx context.Context) error {
	done := make(chan error, 1)
	go func() {
		done <- pm.Stop()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (pm *ProcessManager) Status() model.RuntimeStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	status := model.RuntimeStatus{
		Port:       pm.cfg.Port,
		DataDir:    pm.cfg.DataDir,
		Version:    pm.version,
		LastError:  pm.lastError,
		LastExit:   pm.lastExit,
		BinaryPath: pm.binaryPath,
	}

	if pm.cmd == nil || pm.cmd.Process == nil {
		return status
	}
	if pm.cmd.ProcessState != nil && pm.cmd.ProcessState.Exited() {
		return status
	}

	status.Running = true
	status.PID = pm.cmd.Process.Pid
	status.UptimeSec = int64(time.Since(pm.startedAt).Seconds())
	status.StartedAt = pm.startedAt.Format(time.RFC3339)
	status.Healthy = pm.isHealthy()
	return status
}

func (pm *ProcessManager) Logs(limit int) []string {
	if limit <= 0 {
		limit = 100
	}
	return pm.logs.Last(limit)
}

func (pm *ProcessManager) buildEnv() []string {
	env := append([]string{}, os.Environ()...)
	env = append(env, fmt.Sprintf("PORT=%d", pm.cfg.Port))

	managedLines, err := NewManagedEnvService(pm.cfg.DataDir).RuntimeLines()
	if err != nil {
		return env
	}
	env = append(env, managedLines...)
	return env
}

func (pm *ProcessManager) captureOutput(reader io.Reader, stream string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		pm.logs.Add(fmt.Sprintf("%s: %s", stream, line))

		// Emit log entry
		if pm.logService != nil {
			level := model.LogLevelInfo
			if stream == "stderr" {
				// Check for error/warn keywords in stderr
				if strings.Contains(strings.ToLower(line), "error") {
					level = model.LogLevelError
				} else if strings.Contains(strings.ToLower(line), "warn") {
					level = model.LogLevelWarn
				}
			}
			source := "runtime.stdout"
			if stream == "stderr" {
				source = "runtime.stderr"
			}
			entry := model.LogEntry{
				Level:     level,
				Source:    source,
				Event:     model.EventRuntimeStdout,
				Message:   line,
				Timestamp: time.Now().UTC(),
			}
			_ = pm.logService.Write(entry)
		}
	}
	if err := scanner.Err(); err != nil {
		pm.logs.Add(fmt.Sprintf("%s scan error: %v", stream, err))
	}
}

func (pm *ProcessManager) waitForExit(cmd *exec.Cmd) {
	err := cmd.Wait()

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.lastExit = time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		pm.lastError = err.Error()
		pm.logs.Add(fmt.Sprintf("runtime exited with error: %v", err))

		// Emit lifecycle event with error
		if pm.logService != nil {
			entry := model.LogEntry{
				Level:     model.LogLevelError,
				Source:    model.SourceRuntime,
				Event:     model.EventRuntimeLifecycle,
				Message:   fmt.Sprintf("Node-RED process stopped with error: %s", err.Error()),
				Timestamp: time.Now().UTC(),
			}
			_ = pm.logService.Write(entry)
		}
		return
	}

	pm.lastError = ""
	pm.logs.Add("runtime exited cleanly")

	// Emit lifecycle event
	if pm.logService != nil {
		entry := model.LogEntry{
			Level:     model.LogLevelInfo,
			Source:    model.SourceRuntime,
			Event:     model.EventRuntimeLifecycle,
			Message:   "Node-RED process stopped",
			Timestamp: time.Now().UTC(),
		}
		_ = pm.logService.Write(entry)
	}
}

func (pm *ProcessManager) isHealthy() bool {
	resp, err := pm.httpClient.Get(fmt.Sprintf("http://127.0.0.1:%d/", pm.cfg.Port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 500
}

func (pm *ProcessManager) readInstalledVersion() (string, error) {
	runner := platform.NewRunner()
	output, err := runner.Run(pm.cfg.DataDir, "npm", "ls", "node-red", "--depth=0", "--json")
	if err != nil {
		return "", err
	}

	var payload struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return "", err
	}

	if dep, ok := payload.Dependencies["node-red"]; ok {
		return dep.Version, nil
	}

	return "", nil
}

type ringBuffer struct {
	mu    sync.RWMutex
	lines []string
	size  int
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{
		lines: make([]string, 0, size),
		size:  size,
	}
}

func (rb *ringBuffer) Add(line string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	entry := fmt.Sprintf("%s %s", time.Now().UTC().Format(time.RFC3339), line)
	if len(rb.lines) >= rb.size {
		copy(rb.lines, rb.lines[1:])
		rb.lines[len(rb.lines)-1] = entry
		return
	}
	rb.lines = append(rb.lines, entry)
}

func (rb *ringBuffer) Last(limit int) []string {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if limit > len(rb.lines) {
		limit = len(rb.lines)
	}
	result := make([]string, limit)
	copy(result, rb.lines[len(rb.lines)-limit:])
	return result
}
