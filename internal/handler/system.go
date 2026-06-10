package handler

import (
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/composedof2/nrcc/internal/middleware"
	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
)

// SystemHandler handles system information endpoints
type SystemHandler struct {
	nodeVersion    string
	metricsBuffer  *service.MetricsBuffer
	processManager *service.ProcessManager
	startedAt      time.Time
}

// SetMetricsBuffer wires the MetricsBuffer into the SystemHandler so it can
// serve the /api/system/history endpoint.
func (h *SystemHandler) SetMetricsBuffer(buf *service.MetricsBuffer) {
	h.metricsBuffer = buf
}

// SetProcessManager wires the ProcessManager into the SystemHandler so it can
// serve the /api/runtime/history endpoint.
func (h *SystemHandler) SetProcessManager(pm *service.ProcessManager) {
	h.processManager = pm
}

// NewSystemHandler creates a new system handler. The startedAt timestamp
// captures NRCC process start time and is used to compute uptime in /api/health.
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{
		nodeVersion: getNodeVersion(),
		startedAt:   time.Now(),
	}
}

// GetHealth handles GET /api/health — public (no auth required).
// Returns status:"ok", integer uptime (seconds since process start), and
// restartCount (cumulative durable auto-restart count from ProcessManager).
// uptime always reflects real elapsed time since handler construction; if the
// ProcessManager is not yet wired, only restartCount falls back to 0 (no panic).
func (h *SystemHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	uptime := int(time.Since(h.startedAt).Seconds())
	restarts := 0
	if h.processManager != nil {
		restarts = h.processManager.CumulativeRestarts()
	}
	model.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "ok",
		"uptime":       uptime,
		"restartCount": restarts,
	})
}

// CpuInfo represents CPU statistics
type CpuInfo struct {
	Usage float64 `json:"usage"` // percent 0-100
	Cores int     `json:"cores"`
}

// MemoryInfo represents memory statistics
type MemoryInfo struct {
	Total        uint64  `json:"total"`
	Free         uint64  `json:"free"`
	Used         uint64  `json:"used"`
	UsagePercent float64 `json:"usagePercent"`
}

// DiskInfo represents disk statistics
type DiskInfo struct {
	Total        uint64  `json:"total"`
	Free         uint64  `json:"free"`
	Used         uint64  `json:"used"`
	UsagePercent float64 `json:"usagePercent"`
}

// SystemInfo represents the system information
type SystemInfo struct {
	Platform       string     `json:"platform"`
	Arch           string     `json:"arch"`
	NodeVersion    string     `json:"nodeVersion"`
	Hostname       string     `json:"hostname"`
	Uptime         uint64     `json:"uptime"`
	Cpu            CpuInfo    `json:"cpu"`
	Memory         MemoryInfo `json:"memory"`
	Disk           DiskInfo   `json:"disk"`
	NodeRedVersion string     `json:"nodeRedVersion"`
}

// GetSystemInfo handles GET /api/system/info - protected
func (h *SystemHandler) GetSystemInfo(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	hostname, _ := os.Hostname()

	// Get platform-specific system stats
	uptime, memTotal, memFree := getSystemStats()
	memUsed := memTotal - memFree
	var memPercent float64
	if memTotal > 0 {
		memPercent = float64(memUsed) / float64(memTotal) * 100
	}

	// Get disk info from root filesystem
	diskTotal, diskFree, diskUsed := getDiskInfo("/")
	var diskPercent float64
	if diskTotal > 0 {
		diskPercent = float64(diskUsed) / float64(diskTotal) * 100
	}

	// CPU usage (sampled over 200ms on Linux, 0 on other platforms)
	cpuUsage := getCPUUsage()

	info := SystemInfo{
		Platform:    runtime.GOOS,
		Arch:        runtime.GOARCH,
		NodeVersion: h.nodeVersion,
		Hostname:    hostname,
		Uptime:      uptime,
		Cpu: CpuInfo{
			Usage: cpuUsage,
			Cores: runtime.NumCPU(),
		},
		Memory: MemoryInfo{
			Total:        memTotal,
			Free:         memFree,
			Used:         memUsed,
			UsagePercent: memPercent,
		},
		Disk: DiskInfo{
			Total:        diskTotal,
			Free:         diskFree,
			Used:         diskUsed,
			UsagePercent: diskPercent,
		},
		NodeRedVersion: h.nodeRedVersion(),
	}

	model.RespondJSON(w, http.StatusOK, info)
}

// nodeRedVersion resolves the installed Node-RED version from the process
// manager, falling back to "unknown" when it is unavailable.
func (h *SystemHandler) nodeRedVersion() string {
	if h.processManager == nil {
		return "unknown"
	}
	if v := h.processManager.Version(); v != "" {
		return v
	}
	return "unknown"
}

// GetSystemHistory handles GET /api/system/history — returns recent MetricsSnapshot entries.
// Query param ?n=120 (default 120, max 120) controls how many entries are returned.
func (h *SystemHandler) GetSystemHistory(w http.ResponseWriter, r *http.Request) {
	const defaultN = 120
	const maxN = 120

	n := defaultN
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
			n = parsed
		}
	}
	if n > maxN {
		n = maxN
	}

	snapshots := make([]model.MetricsSnapshot, 0)
	if h.metricsBuffer != nil {
		if recent := h.metricsBuffer.Recent(n); recent != nil {
			snapshots = recent
		}
	}

	model.RespondJSON(w, http.StatusOK, snapshots)
}

// runtimeHistoryPayload is the JSON body returned by GetRuntimeHistory.
type runtimeHistoryPayload struct {
	Events []model.RestartEvent `json:"events"`
	Status model.RuntimeStatus  `json:"status"`
}

// GetRuntimeHistory handles GET /api/runtime/history — returns restart events
// and current runtime status from the ProcessManager.
func (h *SystemHandler) GetRuntimeHistory(w http.ResponseWriter, r *http.Request) {
	events := make([]model.RestartEvent, 0)
	var status model.RuntimeStatus

	if h.processManager != nil {
		if raw := h.processManager.RestartEvents(); raw != nil {
			events = raw
		}
		status = h.processManager.Status()
	}

	payload := runtimeHistoryPayload{
		Events: events,
		Status: status,
	}

	model.RespondJSON(w, http.StatusOK, payload)
}

// getNodeVersion retrieves the Node.js version
func getNodeVersion() string {
	cmd := exec.Command("node", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}
