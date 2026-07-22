package service

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// canonicalBootstrap is the authoritative list of bootstrap env var
// names read via os.Getenv in the nrcc binary. Adding a new
// os.Getenv call in main.go or internal/ without extending this list
// (and .env.example + docs/configuration/env-contract.md) makes
// TestEnvContract_BinaryReadsAllCanonicalNames fail.
//
// Node-RED child env (NODE_RED_*) is verified separately by
// TestEnvContract_RuntimeChildEnv; the binary itself does not call
// os.Getenv for those.
var canonicalBootstrap = []string{
	"PORT",
	"DATA_DIR",
	"JWT_SECRET",
	"NRCC_ENCRYPTION_KEY",
	"NRCC_CORS_ORIGINS",
	"NRCC_CORS_UNSAFE_WILDCARD",
	"NRCC_TRUSTED_PROXIES",
	"EDGE_MODE",
	"NRCC_MANAGE_NODE_RED",
	"NRCC_IMAGE",
	"NRCC_AI_ENABLED",
	"NRCC_AI_PROVIDER",
	"NRCC_AI_ENDPOINT",
	"NRCC_AI_MODEL",
	"NRCC_AI_API_KEY",
	"NRCC_BACKUP_DIR",
	"NRCC_RESTIC_REPO",
	"NRCC_RESTIC_BINARY",
	"NRCC_RESTIC_PASSWORD",
	"NRCC_RESTIC_PASSWORD_FILE",
	"NRCC_RESTIC_CACHE_DIR",
	"NPM_BIN",
}

// canonicalRuntimeChildEnv is the set of NODE_RED_* variables that
// ProcessManager injects into the Node-RED child process. They do not
// appear in os.Getenv calls in the binary; they are read from the
// runtime env map in internal/service/process.go.
var canonicalRuntimeChildEnv = []string{
	"NODE_RED_CMD",
	"NODE_RED_PORT",
	"NODE_RED_USER_DIR",
	"NODE_RED_SETTINGS",
}

// findOsGetenvNames walks Go source under root and returns every
// os.Getenv / os.LookupEnv string literal, minus test-only files.
// Test-only env names follow the convention NRCC_TEST_*
// and NRCC_RESTIC_TEST_BIN; PATH is a Go runtime var, not a contract.
func findOsGetenvNames(t *testing.T, root string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	seen := map[string]bool{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			if info != nil && info.IsDir() && (info.Name() == "node_modules" || info.Name() == ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		inspect := func(body ast.Node) {
			ast.Inspect(body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != "os" {
					return true
				}
				if sel.Sel.Name != "Getenv" && sel.Sel.Name != "LookupEnv" {
					return true
				}
				if len(call.Args) == 0 {
					return true
				}
				lit, ok := call.Args[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				name := strings.Trim(lit.Value, `"`)
				seen[name] = true
				return true
			})
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Body != nil {
					inspect(d.Body)
				}
			case *ast.GenDecl:
				// var … = … at package level may call os.Getenv.
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, v := range vs.Values {
						inspect(v)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return seen
}

// TestEnvContract_BinaryReadsAllCanonicalNames asserts that every
// variable listed in canonicalBootstrap is actually read by the
// binary. This catches contract entries that drift from code (the
// old NRCC_BOOTSTRAP_INTERACTIVE was one such entry).
func TestEnvContract_BinaryReadsAllCanonicalNames(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	got := findOsGetenvNames(t, repoRoot)
	for _, name := range canonicalBootstrap {
		if !got[name] {
			t.Errorf("canonicalBootstrap lists %q but no os.Getenv/os.LookupEnv reads it in the binary — drop from contract or add a caller", name)
		}
	}
}

// TestEnvContract_NoUndocumentedBootstrapReads asserts that the
// only os.Getenv calls in non-test code are for canonicalBootstrap
// names plus a small set of Go-internal / test-only vars. New
// os.Getenv calls must extend canonicalBootstrap in the same commit.
func TestEnvContract_NoUndocumentedBootstrapReads(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	got := findOsGetenvNames(t, repoRoot)

	allow := map[string]bool{}
	for _, name := range canonicalBootstrap {
		allow[name] = true
	}
	for _, name := range canonicalRuntimeChildEnv {
		allow[name] = true
	}
	// PATH is read by tooling tests; not a contract.
	allow["PATH"] = true
	// GO_WANT_HELPER_PROCESS is read by go test internals.
	allow["GO_WANT_HELPER_PROCESS"] = true

	for name := range got {
		if !allow[name] {
			t.Errorf("os.Getenv reads %q but it is not in canonicalBootstrap or canonicalRuntimeChildEnv — add it to the contract (and to .env.example / env-contract.md) in the same commit", name)
		}
	}
}

// TestEnvContract_CanonicalNames is a smoke test that the contract
// itself is non-empty. The real assertions live in the two tests
// above.
func TestEnvContract_CanonicalNames(t *testing.T) {
	if len(canonicalBootstrap) == 0 {
		t.Fatal("canonicalBootstrap must list at least one variable")
	}
	if len(canonicalRuntimeChildEnv) == 0 {
		t.Fatal("canonicalRuntimeChildEnv must list at least one variable")
	}
}