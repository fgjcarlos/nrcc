//go:build windows
// +build windows

package handler

// getDiskInfo retrieves disk statistics for a path (Windows version)
// Returns zero values as fallback; proper Windows implementation would use
// GetDiskFreeSpaceEx or similar Windows API
func getDiskInfo(path string) (total, free, used uint64) {
	// Fallback: return zeros
	return 0, 0, 0
}
