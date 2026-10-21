package workflow

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// This is a bounded contract for the repository's known setup-go inputs,
// not a general-purpose YAML parser. All compiler selectors must be patched.
func TestPatchedGoToolchainContract(t *testing.T) {
	selectors := regexp.MustCompile(`(?m)^\s*go-version:\s*([^\r\n]+)`)
	for _, tc := range []struct {
		file  string
		count int
	}{
		{"pr.yml", 3},
		{"security.yml", 1},
		{"release.yml", 1},
		{"language-policy.yml", 1},
	} {
		t.Run(tc.file, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", tc.file))
			if err != nil {
				t.Fatal(err)
			}
			matches := selectors.FindAllStringSubmatch(string(body), -1)
			if len(matches) != tc.count {
				t.Fatalf("got %d compiler selectors, want %d", len(matches), tc.count)
			}
			matrixReferences := 0
			for _, match := range matches {
				switch strings.TrimSpace(match[1]) {
				case `"1.27.3"`, `["1.27.3"]`:
				case `${{ matrix.go-version }}`:
					matrixReferences++
				default:
					t.Errorf("unpatched or unknown compiler selector: %s", match[1])
				}
			}
			if (tc.file == "pr.yml" && matrixReferences != 1) || (tc.file != "pr.yml" && matrixReferences != 0) {
				t.Errorf("unexpected matrix reference count: %d", matrixReferences)
			}
		})
	}
	t.Run("Dockerfile", func(t *testing.T) {
		body, err := os.ReadFile(filepath.Join(repoRoot, "Dockerfile"))
		if err != nil {
			t.Fatal(err)
		}
		want := "FROM golang:1.27.3-alpine@sha256:f2c872bcee949b7be9c490efd02b48a8a2fbf947cdf533f1c58525ec9d8170fd AS go-builder"
		builders := regexp.MustCompile(`(?m)^FROM golang:[^\r\n]+`).FindAllString(string(body), -1)
		if len(builders) != 1 || builders[0] != want {
			t.Fatalf("unexpected Go builder: %v; want %s", builders, want)
		}
		if !strings.Contains(string(body), "# ref: golang:1.27.3-alpine") {
			t.Error("Go builder reference comment must match the patched compiler")
		}
	})
	t.Run("support-matrix", func(t *testing.T) {
		body, err := os.ReadFile(filepath.Join(repoRoot, "docs", "handbook", "operator", "support-matrix.md"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), "| Go | 1.27.3 (golangci-lint, go test, govulncheck) |") {
			t.Error("documented tested Go compiler must match CI")
		}
	})
}
