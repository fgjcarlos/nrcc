package handler

import (
	"testing"

	"github.com/fgjcarlos/nrcc/internal/service"
)

// TestSystemHandler_NodeRedVersion is the #289 regression: the Node-RED version
// must come from the process manager, not a hardcoded "1.3.5".
func TestSystemHandler_NodeRedVersion(t *testing.T) {
	h := NewSystemHandler()

	// No process manager wired → a clear placeholder, never the old constant.
	if v := h.nodeRedVersion(); v != "unknown" {
		t.Errorf("nil process manager: expected \"unknown\", got %q", v)
	}

	// With a process manager, the version is resolved from the command output
	// (here `echo --version` prints "--version") — proving it is not hardcoded.
	pm := service.NewProcessManager("echo", t.TempDir())
	h.SetProcessManager(pm)

	v := h.nodeRedVersion()
	if v == "1.3.5" {
		t.Error("nodeRedVersion still returns the hardcoded 1.3.5")
	}
	if v == "unknown" || v == "" {
		t.Errorf("expected a resolved version from the process manager, got %q", v)
	}
}
