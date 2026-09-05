//go:build linux

package service

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// MetricsSampler periodically samples coherently scoped resource metrics.
type MetricsSampler struct {
	buffer   *MetricsBuffer
	interval time.Duration
	lastCPU  float64
	mu       sync.Mutex
}

// NewMetricsSampler creates a MetricsSampler that writes into buf at the given interval.
func NewMetricsSampler(buf *MetricsBuffer, interval time.Duration) *MetricsSampler {
	return &MetricsSampler{
		buffer:   buf,
		interval: interval,
	}
}

// Start begins periodic sampling. It collects an initial sample immediately, then
// repeats on every tick. It returns when ctx is cancelled.
func (ms *MetricsSampler) Start(ctx context.Context) {
	ms.sample()

	ticker := time.NewTicker(ms.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ms.sample()
		}
	}
}

// LastCPU returns the CPU percentage captured by the most recent sample.
func (ms *MetricsSampler) LastCPU() float64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.lastCPU
}

// sample collects one snapshot and pushes it to the buffer.
func (ms *MetricsSampler) sample() {
	snap := sampleResources()
	snap.Timestamp = time.Now().UTC().Format(time.RFC3339)
	ms.buffer.Push(snap)

	ms.mu.Lock()
	ms.lastCPU = snap.CPUPercent
	ms.mu.Unlock()
}

// sampleResources collects CPU, memory, and disk metrics from one scope.
func sampleResources() model.MetricsSnapshot {
	metrics := CollectResourceMetrics()
	var memoryPercent, diskPercent float64
	if metrics.MemoryAvailable && metrics.MemoryTotal > 0 {
		memoryPercent = float64(metrics.MemoryUsed) / float64(metrics.MemoryTotal) * 100
	}
	if metrics.DiskAvailable && metrics.DiskTotal > 0 {
		diskPercent = float64(metrics.DiskUsed) / float64(metrics.DiskTotal) * 100
	}

	return model.MetricsSnapshot{
		ResourceScope: metrics.Scope,
		Available:     metrics.CPUAvailable && metrics.MemoryAvailable && metrics.DiskAvailable,
		CPUPercent:    metrics.CPUUsage,
		MemoryPercent: memoryPercent,
		DiskPercent:   diskPercent,
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
	defer func() { _ = f.Close() }()

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
