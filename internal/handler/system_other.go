//go:build darwin || freebsd || openbsd || netbsd || windows
// +build darwin freebsd openbsd netbsd windows

package handler

// getSystemStats retrieves system statistics (generic fallback for non-Linux)
func getSystemStats() (uptime uint64, memTotal, memFree uint64) {
	return 0, 0, 0
}

// getCPUUsage returns 0 on non-Linux platforms
func getCPUUsage() float64 {
	return 0
}
