//go:build linux

package service

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	resourceScopeHost        = "host"
	resourceScopeContainer   = "container"
	resourceScopeUnavailable = "unavailable"
)

// CollectResourceMetrics reports one coherent resource scope. A container is
// never populated with host CPU or memory values: cgroup limits are required
// for those metrics, otherwise the individual metric is marked unavailable.
func CollectResourceMetrics() ResourceMetrics {
	if runningInContainer() {
		metrics, err := collectContainerResources("/sys/fs/cgroup", "/proc/self/cgroup")
		if err != nil {
			return ResourceMetrics{Scope: resourceScopeUnavailable}
		}
		return metrics
	}

	return collectHostResources()
}

func runningInContainer() bool {
	for _, marker := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(marker); err == nil {
			return true
		}
	}
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) == 3 && parts[2] != "/" && parts[2] != "" {
			return true
		}
	}
	return false
}

func collectHostResources() ResourceMetrics {
	metrics := ResourceMetrics{Scope: resourceScopeHost}
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err == nil && info.Totalram > 0 {
		metrics.MemoryAvailable = true
		metrics.MemoryTotal = info.Totalram
		metrics.MemoryFree = info.Freeram
		metrics.MemoryUsed = info.Totalram - info.Freeram
	}
	if total, free, used, ok := diskUsage("/"); ok {
		metrics.DiskAvailable = true
		metrics.DiskTotal, metrics.DiskFree, metrics.DiskUsed = total, free, used
	}
	metrics.CPUAvailable = true
	metrics.CPUUsage = hostCPUPercent()
	metrics.CPUCores = runtime.NumCPU()
	return metrics
}

func collectContainerResources(root, cgroupFile string) (ResourceMetrics, error) {
	if _, err := os.Stat(filepath.Join(root, "cgroup.controllers")); err == nil {
		return collectCgroupV2(root, cgroupFile)
	}
	return collectCgroupV1(root, cgroupFile)
}

func collectCgroupV2(root, cgroupFile string) (ResourceMetrics, error) {
	path, err := cgroupV2Path(cgroupFile)
	if err != nil {
		return ResourceMetrics{}, err
	}
	dir := filepath.Join(root, path)
	metrics := ResourceMetrics{Scope: resourceScopeContainer}

	if limit, err := readLimit(filepath.Join(dir, "memory.max")); err == nil && limit > 0 {
		if usage, err := readUint(filepath.Join(dir, "memory.current")); err == nil {
			metrics.MemoryAvailable = true
			metrics.MemoryTotal = limit
			metrics.MemoryUsed = usage
			if usage < limit {
				metrics.MemoryFree = limit - usage
			}
		}
	}
	if quota, period, err := readCPUMax(filepath.Join(dir, "cpu.max")); err == nil && quota > 0 && period > 0 {
		cores := float64(quota) / float64(period)
		if usage, ok := cgroupV2CPUPercent(filepath.Join(dir, "cpu.stat"), cores); ok {
			metrics.CPUAvailable = true
			metrics.CPUCores = int(math.Ceil(cores))
			metrics.CPUUsage = usage
		}
	}
	if total, free, used, ok := diskUsage("/"); ok {
		metrics.DiskAvailable = true
		metrics.DiskTotal, metrics.DiskFree, metrics.DiskUsed = total, free, used
	}
	return metrics, nil
}

func collectCgroupV1(root, cgroupFile string) (ResourceMetrics, error) {
	paths, err := cgroupV1Paths(cgroupFile)
	if err != nil {
		return ResourceMetrics{}, err
	}
	metrics := ResourceMetrics{Scope: resourceScopeContainer}
	memoryDir := filepath.Join(cgroupV1ControllerRoot(root, "memory"), paths["memory"])
	if limit, err := readUint(filepath.Join(memoryDir, "memory.limit_in_bytes")); err == nil && finiteCgroupV1Limit(limit) {
		if usage, err := readUint(filepath.Join(memoryDir, "memory.usage_in_bytes")); err == nil {
			metrics.MemoryAvailable = true
			metrics.MemoryTotal, metrics.MemoryUsed = limit, usage
			if usage < limit {
				metrics.MemoryFree = limit - usage
			}
		}
	}
	cpuRoot := cgroupV1ControllerRoot(root, "cpu", "cpuacct")
	cpuDir := filepath.Join(cpuRoot, paths["cpu"])
	if quota, err := readInt(filepath.Join(cpuDir, "cpu.cfs_quota_us")); err == nil && quota > 0 {
		if period, err := readUint(filepath.Join(cpuDir, "cpu.cfs_period_us")); err == nil && period > 0 {
			cores := float64(quota) / float64(period)
			cpuAcctRoot := cgroupV1ControllerRoot(root, "cpuacct", "cpu")
			if usage, ok := cgroupV1CPUPercent(filepath.Join(cpuAcctRoot, paths["cpuacct"], "cpuacct.usage"), cores); ok {
				metrics.CPUAvailable = true
				metrics.CPUCores = int(math.Ceil(cores))
				metrics.CPUUsage = usage
			}
		}
	}
	if total, free, used, ok := diskUsage("/"); ok {
		metrics.DiskAvailable = true
		metrics.DiskTotal, metrics.DiskFree, metrics.DiskUsed = total, free, used
	}
	return metrics, nil
}

func cgroupV1ControllerRoot(root string, controllers ...string) string {
	for _, controller := range controllers {
		for _, candidate := range []string{controller, "cpu,cpuacct", "cpuacct,cpu"} {
			path := filepath.Join(root, candidate)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return filepath.Join(root, controllers[0])
}

func cgroupV2Path(file string) (string, error) {
	// #nosec G304 -- file is the fixed /proc/self/cgroup path from CollectResourceMetrics or a test fixture.
	data, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("read cgroup v2 membership: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) == 3 && parts[0] == "0" && parts[1] == "" {
			return strings.TrimPrefix(parts[2], "/"), nil
		}
	}
	return "", fmt.Errorf("cgroup v2 membership not found")
}

func cgroupV1Paths(file string) (map[string]string, error) {
	// #nosec G304 -- file is the fixed /proc/self/cgroup path from CollectResourceMetrics or a test fixture.
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read cgroup v1 membership: %w", err)
	}
	paths := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		for _, controller := range strings.Split(parts[1], ",") {
			paths[controller] = strings.TrimPrefix(parts[2], "/")
		}
	}
	if paths["memory"] == "" || paths["cpu"] == "" || paths["cpuacct"] == "" {
		return nil, fmt.Errorf("required cgroup v1 controllers not found")
	}
	return paths, nil
}

func readCPUMax(path string) (quota, period uint64, err error) {
	// #nosec G304 -- path is derived from the cgroup root and controller membership, or supplied by a test fixture.
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) != 2 || fields[0] == "max" {
		return 0, 0, fmt.Errorf("cpu quota unavailable")
	}
	quota, err = strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	period, err = strconv.ParseUint(fields[1], 10, 64)
	return quota, period, err
}

func readLimit(path string) (uint64, error) {
	// #nosec G304 -- path is derived from the cgroup root and controller membership, or supplied by a test fixture.
	data, err := os.ReadFile(path)
	if err != nil || strings.TrimSpace(string(data)) == "max" {
		return 0, fmt.Errorf("limit unavailable")
	}
	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

func readUint(path string) (uint64, error) {
	// #nosec G304 -- path is derived from the cgroup root and controller membership, or supplied by a test fixture.
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

func readInt(path string) (int64, error) {
	// #nosec G304 -- path is derived from the cgroup root and controller membership, or supplied by a test fixture.
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
}

func finiteCgroupV1Limit(limit uint64) bool {
	// cgroup v1 represents an unlimited memory controller with a value close to
	// the signed 64-bit maximum. Treating it as a container limit would be just
	// as misleading as returning the host total.
	return limit > 0 && limit < 1<<60
}

func cgroupV2CPUPercent(path string, cores float64) (float64, bool) {
	first, err := cgroupV2Usage(path)
	if err != nil || cores <= 0 {
		return 0, false
	}
	start := time.Now()
	time.Sleep(200 * time.Millisecond)
	second, err := cgroupV2Usage(path)
	if err != nil {
		return 0, false
	}
	return boundedPercent(float64(second-first) / float64(time.Since(start).Microseconds()) / cores * 100), true
}

func cgroupV2Usage(path string) (uint64, error) {
	// #nosec G304 -- path is derived from the cgroup root and controller membership, or supplied by a test fixture.
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[0] == "usage_usec" {
			return strconv.ParseUint(fields[1], 10, 64)
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("cgroup v2 cpu usage not found")
}

func cgroupV1CPUPercent(path string, cores float64) (float64, bool) {
	first, err := readUint(path)
	if err != nil || cores <= 0 {
		return 0, false
	}
	start := time.Now()
	time.Sleep(200 * time.Millisecond)
	second, err := readUint(path)
	if err != nil {
		return 0, false
	}
	return boundedPercent(float64(second-first) / float64(time.Since(start).Nanoseconds()) / cores * 100), true
}

func hostCPUPercent() float64 { return cpuPercent() }

func boundedPercent(value float64) float64 {
	return math.Max(0, math.Min(100, value))
}

func diskUsage(path string) (total, free, used uint64, ok bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil || stat.Blocks == 0 || stat.Bsize <= 0 {
		return 0, 0, 0, false
	}
	blockSize := uint64(stat.Bsize)
	total = stat.Blocks * blockSize
	free = stat.Bavail * blockSize
	used = (stat.Blocks - stat.Bfree) * blockSize
	return total, free, used, true
}
