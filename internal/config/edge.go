// Package config exposes deployment-level runtime flags resolved from the
// environment. These are NRCC operational toggles (like edge mode), kept
// separate from Node-RED's own settings.js configuration.
package config

import (
	"os"
	"strconv"
)

// EdgeMode reports whether NRCC runs in edge mode, controlled by the EDGE_MODE
// environment variable (ADR 0002). It defaults to false (disabled) when
// EDGE_MODE is unset or unparseable, so existing non-edge deployments behave
// exactly as before.
func EdgeMode() bool {
	return EdgeModeFrom(os.Getenv("EDGE_MODE"))
}

// EdgeModeFrom parses a raw EDGE_MODE value using Go's standard boolean
// semantics (1, t, T, TRUE, true, True → true; everything else, including the
// empty string and non-boolean words, → false). Edge mode is opt-in and
// conservative: anything not clearly understood as "on" stays off.
func EdgeModeFrom(v string) bool {
	enabled, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return enabled
}
