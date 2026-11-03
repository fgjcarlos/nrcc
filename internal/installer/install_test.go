package installer

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestInstallScriptContract runs the published installer (docs/install.sh)
// against stubbed git, docker and curl executables. It proves the contract the
// public command depends on: portable POSIX shell, release-by-default version
// selection, encryption material prepared before Compose, and a readiness wait
// that reports success only for a healthy stack.
func TestInstallScriptContract(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	script := filepath.Join(filepath.Dir(file), "..", "..", "docs", "install.sh")
	for _, shell := range []string{"/bin/sh", "bash"} {
		t.Run(shell, func(t *testing.T) {
			t.Run("generates key and resolves release", func(t *testing.T) {
				h := newHarness(t, script, shell)
				result := h.run(t, nil)
				if result.err != nil {
					t.Fatalf("installer failed: %v\n%s", result.err, result.output)
				}
				assertKeyFile(t, filepath.Join(h.installDir, ".env"))
				if strings.Contains(result.output, h.key) {
					t.Fatal("installer output exposed generated key")
				}
				if !strings.Contains(h.readLog(), "git clone --depth 1 --branch v9.8.7") {
					t.Fatalf("default did not use release tag:\n%s", h.readLog())
				}
			})
			t.Run("generates key without openssl", func(t *testing.T) {
				h := newHarness(t, script, shell)
				h.removeStub("openssl")
				result := h.run(t, nil)
				if result.err != nil {
					t.Fatalf("installer failed: %v\n%s", result.err, result.output)
				}
				assertKeyFile(t, filepath.Join(h.installDir, ".env"))
				if strings.Contains(result.output, h.key) {
					t.Fatal("installer output exposed generated key")
				}
			})
			t.Run("preserves existing key and explicit environment version", func(t *testing.T) {
				h := newHarness(t, script, shell)
				original := "OTHER=value\nNRCC_ENCRYPTION_KEY=operator-secret\nLAST=line\n"
				h.writeExistingInstall(original)
				result := h.run(t, map[string]string{"NRCC_VERSION": "v1.2.3"})
				if result.err != nil {
					t.Fatalf("installer failed: %v\n%s", result.err, result.output)
				}
				got, err := os.ReadFile(filepath.Join(h.installDir, ".env"))
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != original {
					t.Fatalf(".env changed: %q", got)
				}
				if !strings.Contains(h.readLog(), "git clone --depth 1 --branch v1.2.3") {
					t.Fatalf("version not passed to clone:\n%s", h.readLog())
				}
			})
			t.Run("replaces an empty existing key", func(t *testing.T) {
				h := newHarness(t, script, shell)
				h.writeExistingInstall("OTHER=value\nNRCC_ENCRYPTION_KEY=\nLAST=line\n")
				result := h.run(t, nil)
				if result.err != nil {
					t.Fatalf("installer failed: %v\n%s", result.err, result.output)
				}
				path := filepath.Join(h.installDir, ".env")
				got, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(got), "OTHER=value") || !strings.Contains(string(got), "LAST=line") {
					t.Fatalf("operator lines lost: %q", got)
				}
				if strings.Contains(string(got), "NRCC_ENCRYPTION_KEY=\n") {
					t.Fatalf("empty key kept: %q", got)
				}
				assertKeyFile(t, path)
			})
			t.Run("reinstall replaces the checkout and keeps the operator env", func(t *testing.T) {
				h := newHarness(t, script, shell)
				h.writeExistingInstall("NRCC_ENCRYPTION_KEY=operator-secret\n")
				stale := filepath.Join(h.installDir, "stale-from-previous-version.txt")
				if err := os.WriteFile(stale, []byte("stale"), 0600); err != nil {
					t.Fatal(err)
				}
				result := h.run(t, nil)
				if result.err != nil {
					t.Fatalf("installer failed: %v\n%s", result.err, result.output)
				}
				if _, err := os.Stat(stale); !os.IsNotExist(err) {
					t.Fatal("previous checkout was merged instead of replaced")
				}
				got, err := os.ReadFile(filepath.Join(h.installDir, ".env"))
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != "NRCC_ENCRYPTION_KEY=operator-secret\n" {
					t.Fatalf(".env changed: %q", got)
				}
			})
			t.Run("positional version overrides environment", func(t *testing.T) {
				h := newHarness(t, script, shell)
				result := h.runArgs(t, []string{"v2.3.4"}, map[string]string{"NRCC_VERSION": "v1.2.3"})
				if result.err != nil {
					t.Fatalf("installer failed: %v\n%s", result.err, result.output)
				}
				if !strings.Contains(h.readLog(), "git clone --depth 1 --branch v2.3.4") {
					t.Fatalf("positional version not selected:\n%s", h.readLog())
				}
			})
			t.Run("compose failure prints diagnostics", func(t *testing.T) {
				h := newHarness(t, script, shell)
				h.set("DOCKER_MODE", "up-fail")
				result := h.run(t, nil)
				if result.err == nil || !strings.Contains(result.output, "DIAGNOSTIC LOGS") {
					t.Fatalf("expected failed exit and compose logs, got %v\n%s", result.err, result.output)
				}
			})
			t.Run("readiness timeout prints diagnostics", func(t *testing.T) {
				h := newHarness(t, script, shell)
				h.set("DOCKER_MODE", "unhealthy")
				result := h.run(t, map[string]string{"NRCC_READY_TIMEOUT": "0"})
				if result.err == nil || !strings.Contains(result.output, "DIAGNOSTIC LOGS") {
					t.Fatalf("expected timeout and compose logs, got %v\n%s", result.err, result.output)
				}
			})
		})
	}
}

type harness struct {
	t                                         *testing.T
	script, shell, root, installDir, log, key string
	bin                                       string
	env                                       map[string]string
}

type execution struct {
	output string
	err    error
}

func newHarness(t *testing.T, script, shell string) *harness {
	t.Helper()
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	h := &harness{
		t:          t,
		script:     script,
		shell:      shell,
		root:       root,
		bin:        bin,
		installDir: filepath.Join(root, "install"),
		log:        filepath.Join(root, "calls"),
		key:        key,
		env: map[string]string{
			"PATH":        bin + ":/usr/bin:/bin",
			"HOME":        root,
			"INSTALL_DIR": filepath.Join(root, "install"),
			"CALL_LOG":    filepath.Join(root, "calls"),
			"STUB_KEY":    key,
		},
	}
	h.writeStub("openssl", `#!/bin/sh
printf '%s\n' "$STUB_KEY"
`)
	h.writeStub("git", `#!/bin/sh
printf 'git' >> "$CALL_LOG"; for arg do printf ' %s' "$arg" >> "$CALL_LOG"; done; printf '\n' >> "$CALL_LOG"
[ "$1" = clone ] && mkdir -p "$7"
`)
	h.writeStub("docker", `#!/bin/sh
printf 'docker' >> "$CALL_LOG"; for arg do printf ' %s' "$arg" >> "$CALL_LOG"; done; printf '\n' >> "$CALL_LOG"
case "$1 $2" in
  'info ') exit 0;;
  'compose up') [ "${DOCKER_MODE:-}" != up-fail ] || exit 1;;
  'compose ps') if [ "${3:-}" = -q ]; then echo container-id; else echo 'DIAGNOSTIC PS'; fi;;
  'compose logs') echo 'DIAGNOSTIC LOGS';;
  inspect*) [ "${DOCKER_MODE:-}" = unhealthy ] && echo starting || echo healthy;;
esac
`)
	h.writeStub("curl", `#!/bin/sh
case "$*" in *releases/latest*) echo '{"tag_name":"v9.8.7"}';; *healthz*) [ "${DOCKER_MODE:-}" != unhealthy ];; *) exit 0;; esac
`)
	return h
}

// writeExistingInstall prepares a previous nrcc install so the re-install path
// is exercised: an operator .env plus the compose file that identifies the
// directory as an nrcc install.
func (h *harness) writeExistingInstall(env string) {
	h.t.Helper()
	if err := os.MkdirAll(h.installDir, 0700); err != nil {
		h.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(h.installDir, ".env"), []byte(env), 0600); err != nil {
		h.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(h.installDir, "docker-compose.yml"), []byte("services: {}\n"), 0600); err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) writeStub(name, body string) {
	h.t.Helper()
	if err := os.WriteFile(filepath.Join(h.bin, name), []byte(body), 0700); err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) removeStub(name string) {
	h.t.Helper()
	if err := os.Remove(filepath.Join(h.bin, name)); err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) set(k, v string) { h.env[k] = v }

func (h *harness) run(t *testing.T, extra map[string]string) execution {
	return h.runArgs(t, nil, extra)
}

func (h *harness) runArgs(t *testing.T, args []string, extra map[string]string) execution {
	t.Helper()
	//nolint:gosec // G204: the shell and the script path are test-controlled fixtures.
	cmd := exec.Command(h.shell, append([]string{h.script}, args...)...)
	cmd.Env = cleanEnv(h.env, extra)
	out, err := cmd.CombinedOutput()
	return execution{output: string(out), err: err}
}

func cleanEnv(base, extra map[string]string) []string {
	vars := map[string]string{}
	for k, v := range base {
		vars[k] = v
	}
	for k, v := range extra {
		vars[k] = v
	}
	out := make([]string, 0, len(vars))
	for k, v := range vars {
		out = append(out, k+"="+v)
	}
	return out
}

func (h *harness) readLog() string {
	b, err := os.ReadFile(h.log)
	if err != nil && !os.IsNotExist(err) {
		h.t.Fatal(err)
	}
	return string(b)
}

func assertKeyFile(t *testing.T, path string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^NRCC_ENCRYPTION_KEY=([0-9a-f]{64})$`)
	if !re.Match(b) {
		t.Fatalf("expected 64-hex key in .env, got %q", b)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf(".env mode = %o, want 600", info.Mode().Perm())
	}
}
