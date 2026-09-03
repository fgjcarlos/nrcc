//go:build linux

package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCgroupV2Path(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cgroup")
	if err := os.WriteFile(file, []byte("0::/docker/abc123\n"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := cgroupV2Path(file)
	if err != nil {
		t.Fatalf("cgroupV2Path() error = %v", err)
	}
	if got != "docker/abc123" {
		t.Errorf("cgroupV2Path() = %q, want docker/abc123", got)
	}
}

func TestCgroupV1Paths(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cgroup")
	data := "2:cpu,cpuacct:/docker/abc123\n3:memory:/docker/abc123\n"
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := cgroupV1Paths(file)
	if err != nil {
		t.Fatalf("cgroupV1Paths() error = %v", err)
	}
	for _, controller := range []string{"cpu", "cpuacct", "memory"} {
		if got[controller] != "docker/abc123" {
			t.Errorf("%s path = %q, want docker/abc123", controller, got[controller])
		}
	}
}

func TestReadCPUMax(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantQuota  uint64
		wantPeriod uint64
		wantErr    bool
	}{
		{name: "finite quota", content: "100000 100000\n", wantQuota: 100000, wantPeriod: 100000},
		{name: "unlimited quota", content: "max 100000\n", wantErr: true},
		{name: "malformed quota", content: "not-a-number\n", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "cpu.max")
			if err := os.WriteFile(file, []byte(tt.content), 0600); err != nil {
				t.Fatal(err)
			}
			quota, period, err := readCPUMax(file)
			if (err != nil) != tt.wantErr {
				t.Fatalf("readCPUMax() error = %v, wantErr %t", err, tt.wantErr)
			}
			if quota != tt.wantQuota || period != tt.wantPeriod {
				t.Errorf("readCPUMax() = (%d, %d), want (%d, %d)", quota, period, tt.wantQuota, tt.wantPeriod)
			}
		})
	}
}

func TestReadLimit(t *testing.T) {
	tests := []struct {
		content string
		want    uint64
		wantErr bool
	}{
		{content: "536870912\n", want: 536870912},
		{content: "max\n", wantErr: true},
	}
	for _, tt := range tests {
		file := filepath.Join(t.TempDir(), "memory.max")
		if err := os.WriteFile(file, []byte(tt.content), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := readLimit(file)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("readLimit(%q) = (%d, %v), want (%d, error=%t)", tt.content, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestFiniteCgroupV1Limit(t *testing.T) {
	if !finiteCgroupV1Limit(536_870_912) {
		t.Error("finiteCgroupV1Limit() rejected a real container limit")
	}
	if finiteCgroupV1Limit(9_223_372_036_854_771_712) {
		t.Error("finiteCgroupV1Limit() accepted cgroup v1's unlimited sentinel")
	}
}

func TestCgroupV1ControllerRootSupportsCombinedCPUMount(t *testing.T) {
	root := t.TempDir()
	combined := filepath.Join(root, "cpu,cpuacct")
	if err := os.Mkdir(combined, 0700); err != nil {
		t.Fatal(err)
	}
	if got := cgroupV1ControllerRoot(root, "cpu", "cpuacct"); got != combined {
		t.Errorf("cgroupV1ControllerRoot() = %q, want %q", got, combined)
	}
}

func TestCollectCgroupV2MarksCPUUnavailableWhenUsageCounterCannotBeRead(t *testing.T) {
	tests := []struct {
		name       string
		cpuStat    string
		unreadable bool
	}{
		{name: "unreadable cpu.stat", unreadable: true},
		{name: "malformed cpu.stat", cpuStat: "usage_usec not-a-number\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, cgroupFile, dir := setupCgroupV2(t)
			if tt.unreadable {
				if err := os.Mkdir(filepath.Join(dir, "cpu.stat"), 0700); err != nil {
					t.Fatal(err)
				}
			} else if tt.cpuStat != "" {
				writeCgroupFile(t, filepath.Join(dir, "cpu.stat"), tt.cpuStat)
			}

			metrics, err := collectCgroupV2(root, cgroupFile)
			if err != nil {
				t.Fatalf("collectCgroupV2() error = %v", err)
			}
			if metrics.CPUAvailable || metrics.CPUUsage != 0 || metrics.CPUCores != 0 {
				t.Errorf("CPU metrics = %+v, want unavailable", metrics)
			}
			if !metrics.MemoryAvailable {
				t.Error("memory metrics became unavailable when only the CPU counter failed")
			}
		})
	}
}

func TestCollectCgroupV1MarksCPUUnavailableWhenUsageCounterCannotBeRead(t *testing.T) {
	tests := []struct {
		name       string
		usage      string
		unreadable bool
	}{
		{name: "unreadable cpuacct.usage", unreadable: true},
		{name: "malformed cpuacct.usage", usage: "not-a-number\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, cgroupFile, cpuAcctDir := setupCgroupV1(t)
			if tt.unreadable {
				if err := os.Mkdir(filepath.Join(cpuAcctDir, "cpuacct.usage"), 0700); err != nil {
					t.Fatal(err)
				}
			} else if tt.usage != "" {
				writeCgroupFile(t, filepath.Join(cpuAcctDir, "cpuacct.usage"), tt.usage)
			}

			metrics, err := collectCgroupV1(root, cgroupFile)
			if err != nil {
				t.Fatalf("collectCgroupV1() error = %v", err)
			}
			if metrics.CPUAvailable || metrics.CPUUsage != 0 || metrics.CPUCores != 0 {
				t.Errorf("CPU metrics = %+v, want unavailable", metrics)
			}
			if !metrics.MemoryAvailable {
				t.Error("memory metrics became unavailable when only the CPU counter failed")
			}
		})
	}
}

func TestCollectCgroupsIncludesCPUWhenUsageCounterIsReadable(t *testing.T) {
	t.Run("v2", func(t *testing.T) {
		root, cgroupFile, dir := setupCgroupV2(t)
		writeCgroupFile(t, filepath.Join(dir, "cpu.stat"), "usage_usec 100\n")

		metrics, err := collectCgroupV2(root, cgroupFile)
		if err != nil {
			t.Fatalf("collectCgroupV2() error = %v", err)
		}
		if !metrics.CPUAvailable || metrics.CPUCores != 1 {
			t.Errorf("CPU metrics = %+v, want available with one core", metrics)
		}
	})

	t.Run("v1", func(t *testing.T) {
		root, cgroupFile, cpuAcctDir := setupCgroupV1(t)
		writeCgroupFile(t, filepath.Join(cpuAcctDir, "cpuacct.usage"), "100\n")

		metrics, err := collectCgroupV1(root, cgroupFile)
		if err != nil {
			t.Fatalf("collectCgroupV1() error = %v", err)
		}
		if !metrics.CPUAvailable || metrics.CPUCores != 1 {
			t.Errorf("CPU metrics = %+v, want available with one core", metrics)
		}
	})
}

func setupCgroupV2(t *testing.T) (root, cgroupFile, dir string) {
	t.Helper()
	root = t.TempDir()
	writeCgroupFile(t, filepath.Join(root, "cgroup.controllers"), "cpu memory\n")
	dir = filepath.Join(root, "container")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	writeCgroupFile(t, filepath.Join(dir, "cpu.max"), "100000 100000\n")
	writeCgroupFile(t, filepath.Join(dir, "memory.max"), "536870912\n")
	writeCgroupFile(t, filepath.Join(dir, "memory.current"), "268435456\n")
	cgroupFile = filepath.Join(root, "cgroup")
	writeCgroupFile(t, cgroupFile, "0::/container\n")
	return root, cgroupFile, dir
}

func setupCgroupV1(t *testing.T) (root, cgroupFile, cpuAcctDir string) {
	t.Helper()
	root = t.TempDir()
	cpuDir := filepath.Join(root, "cpu", "container")
	cpuAcctDir = filepath.Join(root, "cpuacct", "container")
	memoryDir := filepath.Join(root, "memory", "container")
	for _, dir := range []string{cpuDir, cpuAcctDir, memoryDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	writeCgroupFile(t, filepath.Join(cpuDir, "cpu.cfs_quota_us"), "100000\n")
	writeCgroupFile(t, filepath.Join(cpuDir, "cpu.cfs_period_us"), "100000\n")
	writeCgroupFile(t, filepath.Join(memoryDir, "memory.limit_in_bytes"), "536870912\n")
	writeCgroupFile(t, filepath.Join(memoryDir, "memory.usage_in_bytes"), "268435456\n")
	cgroupFile = filepath.Join(root, "cgroup")
	writeCgroupFile(t, cgroupFile, "2:cpu:/container\n3:cpuacct:/container\n4:memory:/container\n")
	return root, cgroupFile, cpuAcctDir
}

func writeCgroupFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
