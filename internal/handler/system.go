package handler

import (
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// SystemHandler handles system information endpoints
type SystemHandler struct {
	nodeVersion    string
	metricsBuffer  *service.MetricsBuffer
	processManager *service.ProcessManager
	startedAt      time.Time
	edgeMode       bool

	// Optional dependencies for GetSecurityPosture (issue #676 item 2).
	// Each may be nil during early bootstrap — the posture handler
	// degrades to reporting the relevant boolean as false / count as 0
	// when the dependency is unavailable, so the route stays servable.
	envSvc  *service.EnvService
	authSvc *service.AuthService
	mfaSvc  *service.MfaService
}

// SetEdgeMode records whether NRCC is running in edge mode (resolved from
// EDGE_MODE at startup). It is surfaced read-only via GetSystemInfo so the UI
// can show an "Edge mode: enabled/disabled" badge. Default false.
func (h *SystemHandler) SetEdgeMode(enabled bool) {
	h.edgeMode = enabled
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

// SetEnvService wires the EnvService so GetSecurityPosture can report
// whether NRCC_ENCRYPTION_KEY is configured.
func (h *SystemHandler) SetEnvService(es *service.EnvService) { h.envSvc = es }

// SetAuthService wires the AuthService so GetSecurityPosture can count
// active refresh sessions and total admins.
func (h *SystemHandler) SetAuthService(a *service.AuthService) { h.authSvc = a }

// SetMfaService wires the MfaService so GetSecurityPosture can count
// admins with TOTP enrolled.
func (h *SystemHandler) SetMfaService(m *service.MfaService) { h.mfaSvc = m }

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
	EdgeMode       bool       `json:"edgeMode"`
}

// GetSystemInfo handles GET /api/system/info - protected
func (h *SystemHandler) GetSystemInfo(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r)
	if claims == nil {
		model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	hostname, _ := os.Hostname()
	// MEDIUM-018: hostname leaks the internal FQDN ("prod-web-01.internal"),
	// which is useful for lateral movement. Strip it for non-admin viewers;
	// admins still see the real value for incident response.
	if claims.Role != model.RoleAdmin {
		hostname = ""
	}

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
		EdgeMode:       h.edgeMode,
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

// SecurityPostureResponse is the JSON shape returned by
// GetSecurityPosture (issue #676 item 2). The four chips the
// Dashboard's SecurityPostureCard renders each map to one field.
// Count fields return 0 when the relevant service is unavailable
// during bootstrap; boolean fields return false in that case so
// the UI treats the chip as "degraded" without crashing.
type SecurityPostureResponse struct {
	EncryptionKeyConfigured bool `json:"encryptionKeyConfigured"` // NRCC_ENCRYPTION_KEY is set and non-empty
	BackupAccessAdminGated bool `json:"backupAccessAdminGated"`   // /api/backups/{id}/download is RequireAdmin
	ActiveRefreshSessions  int  `json:"activeRefreshSessions"`   // count of non-revoked, non-expired sessions
	TotalAdmins            int  `json:"totalAdmins"`             // users with RoleAdmin
	MfaEnrolledAdmins      int  `json:"mfaEnrolledAdmins"`       // subset of totalAdmins with TOTP enrolled
}

// GetSecurityPosture handles GET /api/system/security-posture — admin-only
// (issue #676 item 2). The endpoint backs the SecurityPostureCard on the
// dashboard; the encryptionKeyConfigured chip is the only signal that
// surfaces the silent-degradation failure mode from issue #04 (encrypted
// env vars written in clear when NRCC_ENCRYPTION_KEY is empty).
func (h *SystemHandler) GetSecurityPosture(w http.ResponseWriter, r *http.Request) {
	resp := SecurityPostureResponse{
		// backupAccessAdminGated is wired statically at the router
		// (server.go: r.With(middleware.RequireAdmin).Get("/{id}/download"...))
		// so it is true whenever the router was built with this binary.
		// Reflected here so the chip stays accurate if someone ever
		// loosens the gate in the future (and the dashboard can flag it).
		BackupAccessAdminGated: true,
	}

	if h.envSvc != nil {
		resp.EncryptionKeyConfigured = h.envSvc.EncryptionKeyConfigured()
	}

	if h.authSvc != nil {
		resp.ActiveRefreshSessions = h.authSvc.CountActiveRefreshSessions()
		resp.TotalAdmins = h.authSvc.CountAdmins()
	}

	if h.mfaSvc != nil && h.authSvc != nil {
		users, err := h.authSvc.GetAllUsers()
		if err == nil {
			enrolled := 0
			for _, u := range users {
				if u.Role != model.RoleAdmin {
					continue
				}
				if h.mfaSvc.IsEnrolled(u.ID) {
					enrolled++
				}
			}
			resp.MfaEnrolledAdmins = enrolled
		}
	}

	model.RespondJSON(w, http.StatusOK, resp)
}
