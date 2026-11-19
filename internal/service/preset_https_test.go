package service

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// nodeCheck asserts that src is syntactically valid JavaScript. It skips
// when node is not installed so the Go suite stays runnable without it.
func nodeCheck(t *testing.T, src string) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed; syntax check skipped")
	}
	f := filepath.Join(t.TempDir(), "settings.js")
	if err := os.WriteFile(f, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(node, "--check", f).CombinedOutput(); err != nil {
		t.Fatalf("patched settings.js is not valid JavaScript: %v\n%s\n---\n%s", err, out, src)
	}
}

const settingsWithoutHTTPS = "module.exports = {\n  uiPort: 1880,\n  flowFile: 'flows.json',\n}\n"

func TestHTTPSPreset_WithoutPathsAndWithoutBlockNeedsInput(t *testing.T) {
	res, err := NewPresetRegistry().ApplyPreset(settingsWithoutHTTPS, "https-tls-preset", nil)
	if !errors.Is(err, ErrPresetNeedsInput) {
		t.Fatalf("err = %v, want ErrPresetNeedsInput", err)
	}
	if res.After != settingsWithoutHTTPS {
		t.Errorf("After must be the original source on failure, got:\n%s", res.After)
	}
}

func TestHTTPSPreset_WithPathsWritesLoadableBlock(t *testing.T) {
	res, err := NewPresetRegistry().ApplyPreset(settingsWithoutHTTPS, "https-tls-preset", map[string]string{
		"key":  "/data/certs/privkey.pem",
		"cert": "/data/certs/fullchain.pem",
	})
	if err != nil {
		t.Fatalf("ApplyPreset: %v", err)
	}
	for _, want := range []string{
		`key: require("fs").readFileSync("/data/certs/privkey.pem"),`,
		`cert: require("fs").readFileSync("/data/certs/fullchain.pem"),`,
		"uiPort: 1880,",
	} {
		if !strings.Contains(res.After, want) {
			t.Errorf("After missing %q:\n%s", want, res.After)
		}
	}
	if strings.Contains(res.After, "$KEY_PATH") || strings.Contains(res.After, "$CERT_PATH") {
		t.Errorf("After still carries placeholders:\n%s", res.After)
	}
	nodeCheck(t, res.After)
}

func TestHTTPSPreset_ReplacesExistingBlockWithLoadableBlock(t *testing.T) {
	src := "module.exports = {\n  https: {\n    key: require(\"fs\").readFileSync(\"/old/key.pem\"),\n  },\n  uiPort: 1880,\n}\n"
	res, err := NewPresetRegistry().ApplyPreset(src, "https-tls-preset", map[string]string{
		"key":  "/new/key.pem",
		"cert": "/new/cert.pem",
	})
	if err != nil {
		t.Fatalf("ApplyPreset: %v", err)
	}
	if strings.Contains(res.After, "/old/key.pem") || !strings.Contains(res.After, "/new/cert.pem") {
		t.Errorf("https block not replaced:\n%s", res.After)
	}
	if !strings.Contains(res.After, "),\n  },\n  uiPort: 1880,\n}\n") {
		t.Errorf("replaced block must keep the surrounding lines intact:\n%s", res.After)
	}
	nodeCheck(t, res.After)
}

func TestHTTPSPreset_WithoutPathsKeepsExistingBlock(t *testing.T) {
	src := "module.exports = {\n  https: {\n    key: require(\"fs\").readFileSync(\"/etc/key.pem\"),\n    cert: require(\"fs\").readFileSync(\"/etc/cert.pem\"),\n  },\n  uiPort: 1880,\n}\n"
	res, err := NewPresetRegistry().ApplyPreset(src, "https-tls-preset", nil)
	if err != nil {
		t.Fatalf("ApplyPreset: %v", err)
	}
	if res.After != src {
		t.Errorf("existing https block must stay byte-identical, got:\n%s", res.After)
	}
}

func TestHTTPSPreset_RejectsUnsafePaths(t *testing.T) {
	for name, values := range map[string]map[string]string{
		"relative key":     {"key": "certs/key.pem", "cert": "/data/cert.pem"},
		"missing cert":     {"key": "/data/key.pem"},
		"newline in path":  {"key": "/data/key.pem\n}", "cert": "/data/cert.pem"},
		"quote breaks out": {"key": "/data/key.pem\"), x: (\"", "cert": "/data/cert.pem"},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := NewPresetRegistry().ApplyPreset(settingsWithoutHTTPS, "https-tls-preset", values)
			if !errors.Is(err, ErrPresetNeedsInput) {
				t.Fatalf("err = %v, want ErrPresetNeedsInput", err)
			}
			if res.After != settingsWithoutHTTPS {
				t.Errorf("After must be the original source on rejection")
			}
		})
	}
}
