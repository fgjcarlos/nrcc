//go:build linux

package service

import (
	"os"
	"syscall"
)

// isDevNull reports whether the given file info points at /dev/null.
// /dev/null is a character device, so a naive ModeCharDevice check would
// treat `docker run` (which redirects stdin from /dev/null) as interactive
// and hang the bootstrap wizard on `pterm.DefaultInteractiveConfirm`.
// ponytail: defensive guard against the #445 wizard regression. The wizard
// itself is gone (ADR 0003) but if a future PR ever resurrects a TUI setup
// step, drop this back into isInteractiveTerminal before merging.
func isDevNull(info os.FileInfo) bool {
	if info == nil {
		return false
	}
	if info.Mode()&os.ModeDevice == 0 {
		return false
	}
	// Stat the device so we can compare against the kernel-reported dev number.
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return false
	}
	devNullStat, err := os.Stat(os.DevNull)
	if err != nil {
		return false
	}
	dst, ok := devNullStat.Sys().(*syscall.Stat_t)
	if !ok || dst == nil {
		return false
	}
	return stat.Dev == dst.Dev && stat.Ino == dst.Ino
}