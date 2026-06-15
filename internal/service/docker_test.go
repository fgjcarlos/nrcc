package service

import (
	"errors"
	"os/exec"
	"testing"
)

// capturedCmd records every docker invocation and feeds the docker
// ps / inspect / restart / stop responses from the supplied test
// case. The failOn map can force a non-zero exit for specific
// sub-commands (e.g. "restart" or "stop") to exercise the error path.
type capturedCmd struct {
	psOutput      string
	inspectOutput string
	failOn        map[string]bool
	recorded      [][]string
}

func (c *capturedCmd) factory() func(name string, arg ...string) *exec.Cmd {
	return func(name string, arg ...string) *exec.Cmd {
		rec := append([]string{name}, arg...)
		c.recorded = append(c.recorded, rec)
		if len(arg) > 0 && c.failOn[arg[0]] {
			return exec.Command("sh", "-c", "echo docker-failed >&2; exit 1")
		}
		// ps returns a tab-separated body that goes through Output().
		if len(arg) > 0 && arg[0] == "ps" {
			return exec.Command("printf", "%s", c.psOutput)
		}
		// inspect returns a tab-separated body.
		if len(arg) > 0 && arg[0] == "inspect" {
			return exec.Command("printf", "%s", c.inspectOutput)
		}
		// restart / stop / rm are no-ops on success; we just exit 0.
		return exec.Command("true")
	}
}

func newServiceWithDocker(t *testing.T, c *capturedCmd) *DockerService {
	t.Helper()
	return NewDockerService().
		WithLookPath(func(string) (string, error) { return "/usr/bin/docker", nil }).
		WithExecCommand(c.factory())
}

func TestDockerService_Available(t *testing.T) {
	t.Run("missing binary", func(t *testing.T) {
		svc := NewDockerService().WithLookPath(func(string) (string, error) {
			return "", errors.New("not found")
		})
		if svc.Available() {
			t.Fatalf("expected Available=false when docker is missing")
		}
	})

	t.Run("present binary", func(t *testing.T) {
		svc := NewDockerService().WithLookPath(func(string) (string, error) {
			return "/usr/bin/docker", nil
		})
		if !svc.Available() {
			t.Fatalf("expected Available=true when docker is on PATH")
		}
	})

	t.Run("nil lookPath falls back to exec.LookPath", func(t *testing.T) {
		svc := NewDockerService().WithLookPath(nil)
		// We can't make a strong assertion because the host may or
		// may not have docker installed; we only assert that calling
		// Available with a nil shim does NOT panic.
		_ = svc.Available()
	})
}

func TestDockerService_DiscoverNodeRed(t *testing.T) {
	t.Run("missing docker CLI", func(t *testing.T) {
		svc := NewDockerService().WithLookPath(func(string) (string, error) {
			return "", errors.New("not found")
		})
		_, err := svc.DiscoverNodeRed()
		if !errors.Is(err, errDockerNotFound) {
			t.Fatalf("expected errDockerNotFound, got %v", err)
		}
	})

	t.Run("no matching container", func(t *testing.T) {
		c := &capturedCmd{
			psOutput: "abcdef\tredis:latest\tredis\tExited (0) 1 minute ago\t2026-01-01 00:00:00 +0000 UTC\n",
		}
		svc := newServiceWithDocker(t, c)
		_, err := svc.DiscoverNodeRed()
		if !errors.Is(err, errDockerNotFound) {
			t.Fatalf("expected errDockerNotFound, got %v", err)
		}
	})

	t.Run("finds node-red container and reads state", func(t *testing.T) {
		c := &capturedCmd{
			psOutput: "aaa111\tredis:latest\tredis\tExited (0) 1 minute ago\t2026-01-01 00:00:00 +0000 UTC\n" +
				"bbb222\tnodered/node-red:latest\tnrcc-node-red\tUp 5 minutes\t2026-01-02 00:00:00 +0000 UTC\n",
			inspectOutput: "true\tfalse\t3\n",
		}
		svc := newServiceWithDocker(t, c)
		info, err := svc.DiscoverNodeRed()
		if err != nil {
			t.Fatalf("DiscoverNodeRed err: %v", err)
		}
		if info.ID != "bbb222" {
			t.Fatalf("expected id bbb222, got %q", info.ID)
		}
		if info.Name != "nrcc-node-red" {
			t.Fatalf("expected name nrcc-node-red, got %q", info.Name)
		}
		if info.Image != "nodered/node-red:latest" {
			t.Fatalf("expected image nodered/node-red:latest, got %q", info.Image)
		}
		if info.Status != "running" {
			t.Fatalf("expected status running, got %q", info.Status)
		}
		if info.Created == "" {
			t.Fatalf("expected created to be populated")
		}
		if !info.State.Running {
			t.Fatalf("expected State.Running=true")
		}
		if info.State.Paused {
			t.Fatalf("expected State.Paused=false")
		}
		if info.State.RestartCount != 3 {
			t.Fatalf("expected State.RestartCount=3, got %d", info.State.RestartCount)
		}
	})

	t.Run("match by name when image does not contain node-red", func(t *testing.T) {
		c := &capturedCmd{
			psOutput: "ccc333\tcustom-io:latest\tmy-node-red-svc\tUp 2 minutes\t2026-02-01 00:00:00 +0000 UTC\n",
		}
		svc := newServiceWithDocker(t, c)
		info, err := svc.DiscoverNodeRed()
		if err != nil {
			t.Fatalf("DiscoverNodeRed err: %v", err)
		}
		if info.Name != "my-node-red-svc" {
			t.Fatalf("expected match by name, got %q", info.Name)
		}
	})

	t.Run("docker ps failure wraps errDockerNotFound", func(t *testing.T) {
		svc := NewDockerService().
			WithLookPath(func(string) (string, error) { return "/usr/bin/docker", nil }).
			WithExecCommand(func(string, ...string) *exec.Cmd { return exec.Command("false") })
		_, err := svc.DiscoverNodeRed()
		if !errors.Is(err, errDockerNotFound) {
			t.Fatalf("expected errDockerNotFound, got %v", err)
		}
	})
}

func TestDockerService_Status(t *testing.T) {
	t.Run("returns not-available when CLI is missing", func(t *testing.T) {
		svc := NewDockerService().WithLookPath(func(string) (string, error) {
			return "", errors.New("missing")
		})
		got := svc.Status()
		if got.Available {
			t.Fatalf("expected Available=false")
		}
		if got.Source != "docker-cli" {
			t.Fatalf("expected Source=docker-cli, got %q", got.Source)
		}
		if got.Message == "" {
			t.Fatalf("expected a human message")
		}
	})

	t.Run("returns full payload with container", func(t *testing.T) {
		c := &capturedCmd{
			psOutput:      "ddd444\tnodered/node-red:latest\tnrcc-node-red\tUp 1 minute\t2026-03-01 00:00:00 +0000 UTC\n",
			inspectOutput: "true\tfalse\t0\n",
		}
		svc := newServiceWithDocker(t, c)
		got := svc.Status()
		if !got.Available {
			t.Fatalf("expected Available=true")
		}
		if got.InDocker {
			t.Fatalf("expected InDocker=false on the native path")
		}
		if got.Container == nil || got.Container.Name != "nrcc-node-red" {
			t.Fatalf("expected Container.Name=nrcc-node-red, got %+v", got.Container)
		}
	})
}

func TestDockerService_RestartAndStop(t *testing.T) {
	t.Run("restart runs docker restart on the discovered name", func(t *testing.T) {
		c := &capturedCmd{
			psOutput:      "eee555\tnodered/node-red:latest\tnrcc-node-red\tUp 1 minute\t2026-04-01 00:00:00 +0000 UTC\n",
			inspectOutput: "true\tfalse\t0\n",
		}
		svc := newServiceWithDocker(t, c)
		if err := svc.Restart(); err != nil {
			t.Fatalf("Restart err: %v", err)
		}
		// The last recorded call must be `docker restart nrcc-node-red`.
		var last []string
		for _, rec := range c.recorded {
			if len(rec) >= 3 && rec[0] == "docker" && rec[1] == "restart" {
				last = rec
			}
		}
		if last == nil {
			t.Fatalf("expected a `docker restart …` call, recorded=%v", c.recorded)
		}
		if last[2] != "nrcc-node-red" {
			t.Fatalf("expected docker restart nrcc-node-red, got %v", last)
		}
	})

	t.Run("stop runs docker stop on the discovered name", func(t *testing.T) {
		c := &capturedCmd{
			psOutput:      "fff666\tnodered/node-red:latest\tnrcc-node-red\tUp 1 minute\t2026-04-01 00:00:00 +0000 UTC\n",
			inspectOutput: "true\tfalse\t0\n",
		}
		svc := newServiceWithDocker(t, c)
		if err := svc.Stop(); err != nil {
			t.Fatalf("Stop err: %v", err)
		}
		var last []string
		for _, rec := range c.recorded {
			if len(rec) >= 3 && rec[0] == "docker" && rec[1] == "stop" {
				last = rec
			}
		}
		if last == nil {
			t.Fatalf("expected a `docker stop …` call, recorded=%v", c.recorded)
		}
		if last[2] != "nrcc-node-red" {
			t.Fatalf("expected docker stop nrcc-node-red, got %v", last)
		}
	})

	t.Run("action failure when no container is discovered", func(t *testing.T) {
		c := &capturedCmd{
			psOutput: "gggg777\tother:latest\tother\tExited (0)\t2026-04-01\n",
		}
		svc := newServiceWithDocker(t, c)
		err := svc.Stop()
		if !errors.Is(err, errDockerNotFound) {
			t.Fatalf("expected errDockerNotFound, got %v", err)
		}
	})

	t.Run("action failure when docker CLI exits non-zero", func(t *testing.T) {
		c := &capturedCmd{
			psOutput:      "hhhh888\tnodered/node-red:latest\tnrcc-node-red\tUp 1 minute\t2026-04-01 00:00:00 +0000 UTC\n",
			inspectOutput: "true\tfalse\t0\n",
			failOn:        map[string]bool{"restart": true},
		}
		svc := newServiceWithDocker(t, c)
		err := svc.Restart()
		if err == nil {
			t.Fatalf("expected Restart to surface the docker failure")
		}
		if errors.Is(err, errDockerNotFound) {
			t.Fatalf("expected a wrapped action error, got %v", err)
		}
	})
}

func TestMapContainerStatus(t *testing.T) {
	cases := map[string]string{
		"Up 5 minutes":             "running",
		"Exited (0) 3 minutes ago": "exited",
		"Paused (by user)":         "paused",
		"Restarting (1) 2 seconds": "restarting",
		"Created":                  "created",
		"Removing":                 "removing",
		"Dead":                     "dead",
		"garbage":                  "exited",
	}
	for in, want := range cases {
		if got := mapContainerStatus(in); got != want {
			t.Errorf("mapContainerStatus(%q)=%q want %q", in, got, want)
		}
	}
}
