package service

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
)

// Issue #888: the canonical docker-compose.yml must route every variable of
// the env contract deliberately. Each canonical name belongs to exactly one
// of the sets below; the tests fail when a new contract variable is added
// without deciding how Compose handles it.

// composeForwarded are operational settings the host .env (or shell) sets and
// Compose forwards unchanged as "${NAME:-}". An empty value keeps the binary
// default, because every one of these readers treats "" like unset.
var composeForwarded = []string{
	"JWT_SECRET",
	"NRCC_CORS_ORIGINS",
	"NRCC_CORS_UNSAFE_WILDCARD",
	"NRCC_TRUSTED_PROXIES",
	"NRCC_TLS",
	"EDGE_MODE",
	"NRCC_METRICS_PUBLIC",
	"NRCC_BULK_MAX_ENTRIES",
	"NRCC_BULK_MAX_ENTRY_BYTES",
	"NRCC_AI_ENABLED",
	"NRCC_AI_PROVIDER",
	"NRCC_AI_ENDPOINT",
	"NRCC_AI_MODEL",
	"NRCC_AI_API_KEY",
}

// composeRequired are forwarded with the ":?" form, so Compose refuses to
// start when the host does not set them (#664).
var composeRequired = []string{
	"NRCC_ENCRYPTION_KEY",
}

// composePinned are fixed by the one-container image layout: in-container
// ports and paths. Host-side differences between stacks go through
// NRCC_HOST_PORT / NODE_RED_HOST_PORT and the Compose project name instead.
var composePinned = map[string]string{
	"PORT":              "3001",
	"DATA_DIR":          "/data",
	"NODE_RED_CMD":      "node-red",
	"NODE_RED_USER_DIR": "/data",
	"NODE_RED_SETTINGS": "/data/settings.js",
}

// composeNotForwarded are contract variables the canonical stack
// intentionally does not take from the host, with the reason documented in
// docs/configuration/env-contract.md.
var composeNotForwarded = map[string]string{ //nolint:gosec // G101 variable names and reasons, not credentials.
	"NODE_RED_PORT":             "container port 1880 is fixed by the port mapping; use NODE_RED_HOST_PORT",
	"NRCC_MANAGE_NODE_RED":      "false would leave the one-container stack without a Node-RED process",
	"NRCC_IMAGE":                "display label only; the stack builds the image locally",
	"NRCC_BACKUP_DIR":           "pinned to the nrcc_backups volume at /data/backups",
	"NRCC_BACKUP_ROOT":          "pinned to the /data volume",
	"NPM_BIN":                   "the image ships its own npm",
	"NRCC_RESTIC_REPO":          "Restic is deferred (#432); the image ships no restic binary",
	"NRCC_RESTIC_BINARY":        "Restic is deferred (#432)",
	"NRCC_RESTIC_PASSWORD":      "Restic is deferred (#432)",
	"NRCC_RESTIC_PASSWORD_FILE": "Restic is deferred (#432)",
	"NRCC_RESTIC_CACHE_DIR":     "Restic is deferred (#432)",
}

func repoRootDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

var composeEnvLine = regexp.MustCompile(`^\s+([A-Z][A-Z0-9_]*):\s*"(.*)"\s*$`)

// readComposeEnvironment returns the raw (uninterpolated) values of the nrcc
// service environment block in docker-compose.yml.
func readComposeEnvironment(t *testing.T) map[string]string {
	t.Helper()
	f, err := os.Open(filepath.Join(repoRootDir(t), "docker-compose.yml"))
	if err != nil {
		t.Fatalf("open docker-compose.yml: %v", err)
	}
	defer func() { _ = f.Close() }()

	env := map[string]string{}
	inEnv := false
	envIndent := -1
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if trimmed == "environment:" {
			inEnv = true
			envIndent = indent
			continue
		}
		if !inEnv {
			continue
		}
		if indent <= envIndent {
			break
		}
		m := composeEnvLine.FindStringSubmatch(line)
		if m == nil {
			t.Fatalf("docker-compose.yml environment entry is not NAME: \"value\": %q", line)
		}
		env[m[1]] = m[2]
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan docker-compose.yml: %v", err)
	}
	if len(env) == 0 {
		t.Fatal("docker-compose.yml has no environment block")
	}
	return env
}

// TestComposeEnv_EveryContractVariableIsRouted fails when a contract variable
// has no Compose routing decision, or is listed in more than one set.
func TestComposeEnv_EveryContractVariableIsRouted(t *testing.T) {
	count := map[string]int{}
	for _, n := range composeForwarded {
		count[n]++
	}
	for _, n := range composeRequired {
		count[n]++
	}
	for n := range composePinned {
		count[n]++
	}
	for n := range composeNotForwarded {
		count[n]++
	}
	contract := append(append([]string{}, canonicalBootstrap...), canonicalRuntimeChildEnv...)
	known := map[string]bool{}
	for _, n := range contract {
		known[n] = true
		if count[n] != 1 {
			t.Errorf("%s has %d Compose routing decisions, want exactly 1", n, count[n])
		}
	}
	for n := range count {
		if !known[n] {
			t.Errorf("%s is routed for Compose but is not in the env contract", n)
		}
	}
}

// TestComposeEnv_ServiceBlockMatchesRouting checks the raw environment block.
func TestComposeEnv_ServiceBlockMatchesRouting(t *testing.T) {
	env := readComposeEnvironment(t)

	for _, n := range composeForwarded {
		if got, want := env[n], "${"+n+":-}"; got != want {
			t.Errorf("%s: compose value %q, want %q", n, got, want)
		}
	}
	for _, n := range composeRequired {
		if got := env[n]; !strings.HasPrefix(got, "${"+n+":?") {
			t.Errorf("%s: compose value %q, want the required ${%s:?...} form", n, got, n)
		}
	}
	for n, want := range composePinned {
		if got := env[n]; got != want {
			t.Errorf("%s: compose value %q, want pinned %q", n, got, want)
		}
	}
	for n := range composeNotForwarded {
		if v, ok := env[n]; ok {
			t.Errorf("%s is intentionally not forwarded but compose sets it to %q", n, v)
		}
	}
	for n := range env {
		if !slices.Contains(composeForwarded, n) && !slices.Contains(composeRequired, n) {
			if _, ok := composePinned[n]; !ok {
				t.Errorf("compose sets %s, which has no routing decision", n)
			}
		}
	}
}

// resolveComposeEnv runs `docker compose config` with an isolated env file and
// a minimal process environment, returning the container environment Compose
// would create. It skips when the docker CLI is not installed.
func resolveComposeEnv(t *testing.T, envFile string) map[string]string {
	t.Helper()
	out, stderr, err := runComposeConfig(t, envFile)
	if err != nil {
		t.Fatalf("docker compose config: %v\n%s", err, stderr)
	}
	var cfg struct {
		Services map[string]struct {
			Environment map[string]*string `json:"environment"`
		} `json:"services"`
	}
	if err := json.Unmarshal(out, &cfg); err != nil {
		t.Fatalf("decode compose config: %v", err)
	}
	svc, ok := cfg.Services["nrcc"]
	if !ok {
		t.Fatal("compose config has no nrcc service")
	}
	env := map[string]string{}
	for k, v := range svc.Environment {
		if v != nil {
			env[k] = *v
		}
	}
	return env
}

func runComposeConfig(t *testing.T, envFile string) ([]byte, string, error) {
	t.Helper()
	docker, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker CLI not installed")
	}
	if err := exec.Command(docker, "compose", "version").Run(); err != nil { //nolint:gosec // G204 docker path comes from exec.LookPath.
		t.Skip("docker compose plugin not available")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "stack.env")
	if err := os.WriteFile(path, []byte(envFile), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(docker, "compose", //nolint:gosec // G204 docker path from exec.LookPath, args are test-controlled.
		"-f", filepath.Join(repoRootDir(t), "docker-compose.yml"),
		"--env-file", path,
		"-p", "nrcc-env-contract-test",
		"config", "--format", "json")
	// Only the env file may feed interpolation: no host variables leak in.
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "DOCKER_CONFIG=" + dir}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	return out, stderr.String(), err
}

// TestComposeEnv_MissingEncryptionKeyAborts keeps the #664 guarantee: without
// NRCC_ENCRYPTION_KEY, Compose refuses to resolve the stack.
func TestComposeEnv_MissingEncryptionKeyAborts(t *testing.T) {
	_, stderr, err := runComposeConfig(t, "EDGE_MODE=true\n")
	if err == nil {
		t.Fatal("docker compose config succeeded without NRCC_ENCRYPTION_KEY")
	}
	if !strings.Contains(stderr, "NRCC_ENCRYPTION_KEY must be set") {
		t.Errorf("stderr %q does not name the missing NRCC_ENCRYPTION_KEY", stderr)
	}
}

// TestComposeEnv_DefaultsResolveEmpty: with only the required key set, every
// forwarded setting reaches the container empty (binary default), pinned
// values hold, and nothing else is injected.
func TestComposeEnv_DefaultsResolveEmpty(t *testing.T) {
	env := resolveComposeEnv(t, "NRCC_ENCRYPTION_KEY=contract-test-key-a\n")

	for _, n := range composeForwarded {
		v, ok := env[n]
		if !ok {
			t.Errorf("%s missing from the resolved container env", n)
		} else if v != "" {
			t.Errorf("%s = %q, want empty (binary default)", n, v)
		}
	}
	if got := env["NRCC_ENCRYPTION_KEY"]; got != "contract-test-key-a" {
		t.Errorf("NRCC_ENCRYPTION_KEY = %q, want the env-file value", got)
	}
	for n, want := range composePinned {
		if got := env[n]; got != want {
			t.Errorf("%s = %q, want %q", n, got, want)
		}
	}
	var names []string
	for n := range env {
		names = append(names, n)
	}
	sort.Strings(names)
	if want := len(composeForwarded) + len(composeRequired) + len(composePinned); len(names) != want {
		t.Errorf("resolved env has %d variables %v, want %d", len(names), names, want)
	}
}

// TestComposeEnv_NonDefaultValuesReachContainer: host values for forwarded
// settings arrive verbatim; values for intentionally unforwarded settings do
// not reach the container.
func TestComposeEnv_NonDefaultValuesReachContainer(t *testing.T) {
	values := map[string]string{
		"JWT_SECRET":                "contract-test-jwt-secret",
		"NRCC_CORS_ORIGINS":         "https://a.example,https://b.example",
		"NRCC_CORS_UNSAFE_WILDCARD": "true",
		"NRCC_TRUSTED_PROXIES":      "10.0.0.0/8,192.168.1.10",
		"NRCC_TLS":                  "true",
		"EDGE_MODE":                 "true",
		"NRCC_METRICS_PUBLIC":       "true",
		"NRCC_BULK_MAX_ENTRIES":     "50",
		"NRCC_BULK_MAX_ENTRY_BYTES": "4096",
		"NRCC_AI_ENABLED":           "true",
		"NRCC_AI_PROVIDER":          "openai",
		"NRCC_AI_ENDPOINT":          "https://ai.example/v1",
		"NRCC_AI_MODEL":             "contract-model",
		"NRCC_AI_API_KEY":           "contract-test-ai-key",
	}
	var b strings.Builder
	b.WriteString("NRCC_ENCRYPTION_KEY=contract-test-key-b\n")
	for k, v := range values {
		b.WriteString(k + "=" + v + "\n")
	}
	for n := range composeNotForwarded {
		b.WriteString(n + "=host-value-must-not-leak\n")
	}

	env := resolveComposeEnv(t, b.String())

	for n, want := range values {
		if got := env[n]; got != want {
			t.Errorf("%s = %q, want %q", n, got, want)
		}
	}
	for n := range composeNotForwarded {
		if v, ok := env[n]; ok {
			t.Errorf("%s reached the container as %q; it is intentionally not forwarded", n, v)
		}
	}
	for n, want := range composePinned {
		if got := env[n]; got != want {
			t.Errorf("%s = %q, want pinned %q", n, got, want)
		}
	}
}
