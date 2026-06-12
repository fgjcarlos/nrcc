package config

import "testing"

func TestEdgeModeFrom_TruthyValues(t *testing.T) {
	// Standard Go boolean truthy strings enable edge mode.
	on := []string{"true", "TRUE", "True", "1", "t", "T"}
	for _, v := range on {
		if !EdgeModeFrom(v) {
			t.Errorf("EdgeModeFrom(%q) = false, want true", v)
		}
	}
}

func TestEdgeModeFrom_EverythingElseIsOff(t *testing.T) {
	// Edge mode is opt-in and conservative: anything we do not clearly
	// understand as "on" (including non-Go-boolean words) stays off.
	off := []string{"", "false", "FALSE", "0", "f", "no", "yes", "on", "off", "garbage", "2", " true "}
	for _, v := range off {
		if EdgeModeFrom(v) {
			t.Errorf("EdgeModeFrom(%q) = true, want false", v)
		}
	}
}

func TestEdgeMode_DefaultsFalseWhenUnset(t *testing.T) {
	t.Setenv("EDGE_MODE", "")
	if EdgeMode() {
		t.Errorf("EdgeMode() with EDGE_MODE unset = true, want false (unchanged default)")
	}
}

func TestEdgeMode_ReadsEnv(t *testing.T) {
	t.Setenv("EDGE_MODE", "true")
	if !EdgeMode() {
		t.Errorf("EdgeMode() with EDGE_MODE=true = false, want true")
	}
}
