package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// capturedCmd records every docker invocation and feeds canned
// responses. The DockerService is built with WithExecCommand pointing
// at this factory, so handler tests don't need to touch the service
// package's internals.
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
		if len(arg) > 0 && arg[0] == "ps" {
			return exec.Command("printf", "%s", c.psOutput)
		}
		if len(arg) > 0 && arg[0] == "inspect" {
			return exec.Command("printf", "%s", c.inspectOutput)
		}
		return exec.Command("true")
	}
}

func newDockerNoopAudit(t *testing.T) *audit.Service {
	t.Helper()
	svc, err := audit.NewService(t.TempDir())
	if err != nil {
		t.Fatalf("audit.NewService: %v", err)
	}
	return svc
}

func newDockerServiceWith(t *testing.T, c *capturedCmd) *service.DockerService {
	t.Helper()
	return service.NewDockerService().
		WithLookPath(func(string) (string, error) { return "/usr/bin/docker", nil }).
		WithExecCommand(c.factory())
}

func TestDockerHandler_GetStatus_NativeNotAvailable(t *testing.T) {
	h := NewDockerHandler()
	// Missing CLI: empty PS body → service reports not-available.
	c := &capturedCmd{psOutput: ""}
	h.SetDockerService(newDockerServiceWith(t, c))

	req := httptest.NewRequest("GET", "/api/docker/status", nil)
	w := httptest.NewRecorder()
	h.GetStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data["available"] != false {
		t.Fatalf("expected data.available=false, got %v", resp.Data["available"])
	}
}

func TestDockerHandler_GetStatus_NativeWithContainer(t *testing.T) {
	h := NewDockerHandler()
	c := &capturedCmd{
		psOutput:      "iii999	nodered/node-red:latest	nrcc-node-red	Up 1 minute	2026-05-01 00:00:00 +0000 UTC\n",
		inspectOutput: "true	false	2\n",
	}
	h.SetDockerService(newDockerServiceWith(t, c))
	h.SetAuditService(newDockerNoopAudit(t))

	req := httptest.NewRequest("GET", "/api/docker/status", nil)
	w := httptest.NewRecorder()
	h.GetStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data["name"] != "nrcc-node-red" {
		t.Fatalf("expected data.name=nrcc-node-red, got %v", resp.Data["name"])
	}
	if resp.Data["status"] != "running" {
		t.Fatalf("expected data.status=running, got %v", resp.Data["status"])
	}
	state, _ := resp.Data["state"].(map[string]interface{})
	if state == nil {
		t.Fatalf("expected a state block, got %v", resp.Data)
	}
	if state["running"] != true {
		t.Fatalf("expected state.running=true, got %v", state)
	}
	if rc, _ := state["restartCount"].(float64); rc != 2 {
		t.Fatalf("expected state.restartCount=2, got %v", state["restartCount"])
	}
}

func TestDockerHandler_GetInfo_NativeWithContainer(t *testing.T) {
	h := NewDockerHandler()
	c := &capturedCmd{
		psOutput:      "jjj000	nodered/node-red:latest	nrcc-node-red	Up 1 minute	2026-05-01 00:00:00 +0000 UTC\n",
		inspectOutput: "true	false	0\n",
	}
	h.SetDockerService(newDockerServiceWith(t, c))
	h.SetAuditService(newDockerNoopAudit(t))

	req := httptest.NewRequest("GET", "/api/docker/info", nil)
	w := httptest.NewRecorder()
	h.GetInfo(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data["available"] != true {
		t.Fatalf("expected data.available=true, got %v", resp.Data)
	}
	if resp.Data["containerName"] != "nrcc-node-red" {
		t.Fatalf("expected data.containerName=nrcc-node-red, got %v", resp.Data)
	}
	if resp.Data["source"] != "docker-cli" {
		t.Fatalf("expected data.source=docker-cli, got %v", resp.Data)
	}
}

func TestDockerHandler_PostRestart_NativeMissingService(t *testing.T) {
	h := NewDockerHandler()
	h.SetAuditService(newDockerNoopAudit(t))
	req := httptest.NewRequest("POST", "/api/docker/restart", nil)
	w := httptest.NewRecorder()
	h.PostRestart(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestDockerHandler_PostStop_NativeMissingService(t *testing.T) {
	h := NewDockerHandler()
	h.SetAuditService(newDockerNoopAudit(t))
	req := httptest.NewRequest("POST", "/api/docker/stop", nil)
	w := httptest.NewRecorder()
	h.PostStop(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestDockerHandler_PostRestart_NativeSuccess(t *testing.T) {
	h := NewDockerHandler()
	c := &capturedCmd{
		psOutput:      "kkk111	nodered/node-red:latest	nrcc-node-red	Up 1 minute	2026-05-01 00:00:00 +0000 UTC\n",
		inspectOutput: "true	false	0\n",
	}
	h.SetDockerService(newDockerServiceWith(t, c))
	h.SetAuditService(newDockerNoopAudit(t))

	req := httptest.NewRequest("POST", "/api/docker/restart", nil)
	w := httptest.NewRecorder()
	h.PostRestart(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	var last []string
	for _, rec := range c.recorded {
		if len(rec) >= 3 && rec[0] == "docker" && rec[1] == "restart" {
			last = rec
		}
	}
	if last == nil || last[2] != "nrcc-node-red" {
		t.Fatalf("expected docker restart nrcc-node-red, got %v", c.recorded)
	}
}

func TestDockerHandler_PostStop_NativeSuccess(t *testing.T) {
	h := NewDockerHandler()
	c := &capturedCmd{
		psOutput:      "lll222	nodered/node-red:latest	nrcc-node-red	Up 1 minute	2026-05-01 00:00:00 +0000 UTC\n",
		inspectOutput: "true	false	0\n",
	}
	h.SetDockerService(newDockerServiceWith(t, c))
	h.SetAuditService(newDockerNoopAudit(t))

	req := httptest.NewRequest("POST", "/api/docker/stop", nil)
	w := httptest.NewRecorder()
	h.PostStop(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	var last []string
	for _, rec := range c.recorded {
		if len(rec) >= 3 && rec[0] == "docker" && rec[1] == "stop" {
			last = rec
		}
	}
	if last == nil || last[2] != "nrcc-node-red" {
		t.Fatalf("expected docker stop nrcc-node-red, got %v", c.recorded)
	}
}
