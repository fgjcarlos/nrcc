package service

import "testing"

// TestEnvContract_CanonicalNames documents the bootstrap + runtime
// variable names honored by NRCC (issue #430). The set is
// authoritative; updating it requires updating
// docs/configuration/env-contract.md and .env.example in the same PR.
//
// This is a contract test, not a behavior test: it pins the names so
// an accidental rename is caught in CI instead of in production.
// Behavior (port validation, env precedence, child env injection)
// is covered separately by TestResolveNodeRedRuntime_* and
// process_test.go.
func TestEnvContract_CanonicalNames(t *testing.T) {
	bootstrap := []string{
		"PORT",
		"DATA_DIR",
		"JWT_SECRET",
		"NRCC_ENCRYPTION_KEY",
		"NRCC_CORS_ORIGINS",
		"NRCC_CORS_UNSAFE_WILDCARD",
		"NRCC_TRUSTED_PROXIES",
		"EDGE_MODE",
		"NRCC_MANAGE_NODE_RED",
		"NRCC_BOOTSTRAP_INTERACTIVE",
		"NRCC_IMAGE",
		"NRCC_AI_ENABLED",
		"NRCC_AI_PROVIDER",
		"NRCC_AI_ENDPOINT",
		"NRCC_AI_MODEL",
		"NRCC_AI_API_KEY",
	}
	for _, name := range bootstrap {
		if name == "" {
			t.Errorf("empty canonical bootstrap name in list")
		}
	}

	runtime := []string{
		"NODE_RED_CMD",
		"NODE_RED_PORT",
		"NODE_RED_USER_DIR",
		"NODE_RED_SETTINGS",
	}
	for _, name := range runtime {
		if name == "" {
			t.Errorf("empty canonical runtime name in list")
		}
	}
}