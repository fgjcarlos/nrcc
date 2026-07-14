//go:build linux
// +build linux

package handler

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// getSystemStats retrieves system statistics (Linux version)
func getSystemStats() (uptime uint64, memTotal, memFree uint64) {
	var sysinfo syscall.Sysinfo_t
	if err := syscall.Sysinfo(&sysinfo); err != nil {
		return 0, 0, 0
	}
	return uint64(sysinfo.Uptime), uint64(sysinfo.Totalram), uint64(sysinfo.Freeram)
}

// readCPUStat reads total and idle CPU jiffies from /proc/stat
func readCPUStat() (total, idle uint64) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		// fields: cpu user nice system idle iowait irq softirq steal guest guest_nice
		for i := 1; i < len(fields); i++ {
			v, _ := strconv.ParseUint(fields[i], 10, 64)
			total += v
			if i == 4 { // idle
				idle = v
			}
		}
		return total, idle
	}
	return 0, 0
}

// getCPUUsage samples CPU usage over 200ms
func getCPUUsage() float64 {
	t1, i1 := readCPUStat()
	time.Sleep(200 * time.Millisecond)
	t2, i2 := readCPUStat()

	totalDelta := t2 - t1
	idleDelta := i2 - i1
	if totalDelta == 0 {
		return 0
	}
	return (1.0 - float64(idleDelta)/float64(totalDelta)) * 100
}
