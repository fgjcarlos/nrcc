//go:build linux
// +build linux

package handler

import (
	"syscall"
)

// getSystemStats retrieves system statistics (Linux version)
func getSystemStats() (uptime uint64, memTotal, memFree uint64) {
	var sysinfo syscall.Sysinfo_t
	if err := syscall.Sysinfo(&sysinfo); err != nil {
		return 0, 0, 0
	}
	// #nosec G115 -- 64-bit platforms only; uptime is non-negative since boot.
	return uint64(sysinfo.Uptime), sysinfo.Totalram, sysinfo.Freeram
}
