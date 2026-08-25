//go:build linux

package handler

import "syscall"

// computeDiskUsage derives the (total, free, used) byte counts from a Statfs
// snapshot. Pure so the formula is unit-testable without an actual filesystem.
//
// `used` is total minus *actual* free blocks (Bfree), NOT Bavail: on ext4/xfs
// the kernel reserves ~5% of blocks for root by default, so Bavail < Bfree
// and using Bavail would over-report usage by exactly that reservation.
// `free` keeps Bavail because that's the column `df -h` shows as
// "available". See issue #705.
func computeDiskUsage(blocks, bfree, bavail, bsize uint64) (total, free, used uint64) {
	total = blocks * bsize
	free = bavail * bsize
	used = (blocks - bfree) * bsize
	return total, free, used
}

// getDiskInfo retrieves disk statistics for a path (Unix version)
func getDiskInfo(path string) (total, free, used uint64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0
	}
	// #nosec G115 -- 64-bit platforms only; stat fields are uint64 on every supported target.
	return computeDiskUsage(stat.Blocks, stat.Bfree, stat.Bavail, uint64(stat.Bsize))
}
