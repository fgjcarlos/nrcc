//go:build linux

package service

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/composedof2/nrcc/internal/model"
)

// sampleHost collects CPU, memory, and disk metrics on Linux.
// CPU is measured by reading /proc/stat twice with a 200 ms sleep to compute a delta.
func sampleHost() model.MetricsSnapshot {
	cpu := cpuPercent()
	mem := memPercent()
	disk := diskPercent("/")

	return model.MetricsSnapshot{
		CPUPercent:    cpu,
		MemoryPercent: mem,
		DiskPercent:   disk,
	}
}

// cpuPercent returns the overall CPU utilisation between two /proc/stat reads.
func cpuPercent() float64 {
	t1, err1 := readCPUStat()
	if err1 != nil {
		return 0
	}

	time.Sleep(200 * time.Millisecond)

	t2, err2 := readCPUStat()
	if err2 != nil {
		return 0
	}

	totalDelta := t2.total - t1.total
	idleDelta := t2.idle - t1.idle

	if totalDelta == 0 {
		return 0
	}

	busy := float64(totalDelta-idleDelta) / float64(totalDelta) * 100.0
	if busy < 0 {
		busy = 0
	}
	if busy > 100 {
		busy = 100
	}
	return busy
}

type cpuStat struct {
	total uint64
	idle  uint64
}

func readCPUStat() (cpuStat, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuStat{}, fmt.Errorf("open /proc/stat: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}

		fields := strings.Fields(line)
		// fields[0] = "cpu", [1]=user [2]=nice [3]=system [4]=idle [5]=iowait ...
		if len(fields) < 5 {
			return cpuStat{}, fmt.Errorf("unexpected /proc/stat format")
		}

		var values [8]uint64
		for i := 1; i <= 8 && i < len(fields); i++ {
			v, err := strconv.ParseUint(fields[i], 10, 64)
			if err != nil {
				return cpuStat{}, fmt.Errorf("parse /proc/stat field %d: %w", i, err)
			}
			values[i-1] = v
		}

		// idle = idle + iowait (indices 3 and 4 in values, 0-based)
		idle := values[3] + values[4]
		total := values[0] + values[1] + values[2] + values[3] + values[4] + values[5] + values[6] + values[7]
		return cpuStat{total: total, idle: idle}, nil
	}

	return cpuStat{}, fmt.Errorf("/proc/stat: cpu line not found")
}

// memPercent returns used memory as a percentage of total RAM.
func memPercent() float64 {
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return 0
	}
	if info.Totalram == 0 {
		return 0
	}
	used := info.Totalram - info.Freeram - info.Bufferram - info.Sharedram
	return float64(used) / float64(info.Totalram) * 100.0
}

// diskPercent returns used disk space as a percentage of total space for the given path.
func diskPercent(path string) float64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0
	}
	if stat.Blocks == 0 {
		return 0
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	if total == 0 {
		return 0
	}
	return float64(total-free) / float64(total) * 100.0
}
