//go:build linux

package handler

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
