package handler

import (
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/composedof2/nrcc/internal/middleware"
	"github.com/composedof2/nrcc/internal/model"
)

// SystemHandler handles system information endpoints
type SystemHandler struct {
	nodeVersion string
}

// NewSystemHandler creates a new system handler
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{
		nodeVersion: getNodeVersion(),
	}
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
		NodeRedVersion: "1.3.5",
	}

	model.RespondJSON(w, http.StatusOK, info)
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
