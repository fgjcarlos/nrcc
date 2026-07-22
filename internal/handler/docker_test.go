package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

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

func newDockerServiceWith(t *testing.T, c *capturedCmd) *service.DockerService {
	t.Helper()
	return service.NewDockerService().
		WithLookPath(func(string) (string, error) { return "/usr/bin/docker", nil }).
		WithExecCommand(c.factory())
}

// TestDockerHandler_GetStatus_NativeNotAvailable covers the case where
// the docker CLI is missing or reports no containers. The handler
// returns the not-available payload.
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

// TestDockerHandler_GetStatus_NativeWithContainer covers the happy
// path: docker reports a Node-RED container, the handler returns its
// parsed state.
func TestDockerHandler_GetStatus_NativeWithContainer(t *testing.T) {
	h := NewDockerHandler()
	c := &capturedCmd{
		psOutput:      "iii999\tnodered/node-red:5.0.1-minimal\tnrcc-node-red\tUp 1 minute\t2026-05-01 00:00:00 +0000 UTC\n",
		inspectOutput: "true\tfalse\t2\n",
	}
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
