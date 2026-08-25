//go:build linux

package handler

import "testing"

// TestComputeDiskUsage_NoReservedBlocks covers the happy path: Bavail == Bfree
// (filesystem with no reservation, e.g. tmpfs / fat), so used must be
// total - free. Regression guard for the historical formula.
func TestComputeDiskUsage_NoReservedBlocks(t *testing.T) {
	const bsize = 4096
	total, free, used := computeDiskUsage(1000, 400, 400, bsize)

	if want := uint64(1000 * bsize); total != want {
		t.Fatalf("total = %d, want %d", total, want)
	}
	if want := uint64(400 * bsize); free != want {
		t.Fatalf("free = %d, want %d", free, want)
	}
	if want := uint64(600 * bsize); used != want {
		t.Fatalf("used = %d, want %d", used, want)
	}
}

// TestComputeDiskUsage_Ext4ReservedBlocks is the regression test for #705:
// on ext4/xfs with the default 5% reservation, Bavail is smaller than Bfree.
// used MUST be computed from Bfree (total - Bfree), not from Bavail — using
// Bavail over-reports usage by exactly the reservation.
//
// Numbers below model a 1 TiB ext4: 268435456 blocks × 4096 B, 5% reserved
// (13421773 blocks), so Bfree = 200000000 and Bavail = 186578227.
func TestComputeDiskUsage_Ext4ReservedBlocks(t *testing.T) {
	const (
		bsize  = uint64(4096)
		blocks = uint64(268_435_456)
		bfree  = uint64(200_000_000)
		bavail = uint64(186_578_227) // Bfree minus reserved blocks
	)

	total, free, used := computeDiskUsage(blocks, bfree, bavail, bsize)

	if want := blocks * bsize; total != want {
		t.Fatalf("total = %d, want %d", total, want)
	}
	if want := bavail * bsize; free != want {
		t.Fatalf("free = %d, want %d (Bavail × bsize; the column df -h reports)", free, want)
	}
	// used = (Blocks - Bfree) * Bsize, NOT total - free.
	// With reserved blocks, (Blocks - Bfree) < (Blocks - Bavail); the old
	// formula would have reported 336 GB used instead of the correct 280 GB.
	wantUsed := (blocks - bfree) * bsize
	if used != wantUsed {
		t.Fatalf("used = %d, want %d (Blocks-Bfree × bsize; #705)", used, wantUsed)
	}
	// Sanity: the reserved-block over-report is exactly (Bfree - Bavail)*bsize.
	if (free+used) >= total {
		t.Fatalf("free + used (%d) must be < total (%d): reserved blocks lie between them",
			free+used, total)
	}
	if got := total - free - used; got != (bfree-bavail)*bsize {
		t.Fatalf("gap total - free - used = %d, want reserved-blocks bytes %d",
			got, (bfree-bavail)*bsize)
	}
}