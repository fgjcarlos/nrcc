package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests for the language-policy scanner shipped with issue #768.
//
// The scanner walks a repository and reports suspected non-English
// artefacts in non-frontend files. Spanish sentences in user-visible
// UI strings are out of scope (deferred to issue #767 i18n catalogs),
// so the scanner must skip frontend/src and only flag comments / dev
// messages / docs.

// writeFile creates a file with content under root.
func writeFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	full := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
}

// withTempRepo runs fn against a fresh temporary repository.
func withTempRepo(t *testing.T, fn func(root string)) {
	t.Helper()
	dir := t.TempDir()
	fn(dir)
}

// TestScan_SpanishCommentInGo — happy path: a Spanish sentence in a
// Go source comment is reported as a finding.
func TestScan_SpanishCommentInGo(t *testing.T) {
	withTempRepo(t, func(root string) {
		writeFile(t, root, "internal/foo/bar.go", `package foo

// Esta función valida el usuario antes de continuar.
func Bar() {}
`)
		findings, err := Scan(root)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if len(findings) == 0 {
			t.Fatalf("expected at least one finding, got none")
		}
		var got *Finding
		for i := range findings {
			if findings[i].Path == filepath.Join("internal", "foo", "bar.go") {
				got = &findings[i]
				break
			}
		}
		if got == nil {
			t.Fatalf("expected finding for internal/foo/bar.go, got %+v", findings)
		}
		if !strings.Contains(got.Text, "Esta función") {
			t.Errorf("finding text should contain the Spanish sentence, got %q", got.Text)
		}
	})
}

// TestScan_SpanishInFrontendUIExcluded — Spanish UI strings in
// frontend/src are deferred to issue #767 and must NOT be reported.
func TestScan_SpanishInFrontendUIExcluded(t *testing.T) {
	withTempRepo(t, func(root string) {
		writeFile(t, root, "frontend/src/features/auth/LoginView.tsx", `export function LoginView() {
  return <p>Aún no se ha configurado el usuario administrador.</p>
}
`)
		findings, err := Scan(root)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		for _, f := range findings {
			if strings.Contains(f.Path, "frontend/src") {
				t.Errorf("frontend/src must be excluded, got finding %+v", f)
			}
		}
	})
}

// TestScan_TechnicalUnicodeNotFlagged — math and section symbols in
// comments are legitimate technical Unicode and must not be flagged.
func TestScan_TechnicalUnicodeNotFlagged(t *testing.T) {
	withTempRepo(t, func(root string) {
		writeFile(t, root, "internal/foo/math.go", `package foo

// Tolerance is ±1 step window for clock drift.
// See RFC 9580 §5.4 for the canonical chunked-AEAD layout.
func Drift() int { return 0 }
`)
		findings, err := Scan(root)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		for _, f := range findings {
			if strings.Contains(f.Path, "internal/foo/math.go") {
				t.Errorf("technical Unicode must not be flagged, got %+v", f)
			}
		}
	})
}

// TestScan_LocaleFileExcluded — locale JSON catalogs (when present)
// carry translated UI strings and are exempt.
func TestScan_LocaleFileExcluded(t *testing.T) {
	withTempRepo(t, func(root string) {
		writeFile(t, root, "frontend/src/locales/es.json", `{
  "auth": {
    "notConfigured": "Aún no se ha configurado el usuario administrador."
  }
}
`)
		writeFile(t, root, "frontend/src/locales/en.json", `{
  "auth": {
    "notConfigured": "Administrator user has not been configured yet."
  }
}
`)
		findings, err := Scan(root)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		for _, f := range findings {
			if strings.Contains(f.Path, "frontend/src/locales") {
				t.Errorf("locale files must be excluded, got finding %+v", f)
			}
		}
	})
}

// TestScan_SpanishInMarkdownDocs — Spanish in non-frontend docs is
// reported (markdown has no UI vs source distinction).
func TestScan_SpanishInMarkdownDocs(t *testing.T) {
	withTempRepo(t, func(root string) {
		writeFile(t, root, "CONTRIBUTING.md", `# Contributing

Esta sección describe cómo contribuir al proyecto.
`)
		findings, err := Scan(root)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		found := false
		for _, f := range findings {
			if strings.Contains(f.Path, "CONTRIBUTING.md") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected finding for CONTRIBUTING.md, got %+v", findings)
		}
	})
}

// TestScan_EmptyRepo — no findings in a clean repo.
func TestScan_EmptyRepo(t *testing.T) {
	withTempRepo(t, func(root string) {
		findings, err := Scan(root)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if len(findings) != 0 {
			t.Errorf("clean repo should produce 0 findings, got %d: %+v", len(findings), findings)
		}
	})
}

// TestScan_ExcludedDirs — node_modules, vendor, dist, generated files
// are silently skipped.
func TestScan_ExcludedDirs(t *testing.T) {
	withTempRepo(t, func(root string) {
		writeFile(t, root, "node_modules/foo/bar.go", `package foo
// Esta función está en node_modules.
func Bar() {}
`)
		writeFile(t, root, "vendor/github.com/x/y.go", `package y
// Esta función está en vendor.
func Y() {}
`)
		writeFile(t, root, "frontend/dist/assets/index.js", `// Esta función está en dist.
function x() {}
`)
		findings, err := Scan(root)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		for _, f := range findings {
			if strings.Contains(f.Path, "node_modules") ||
				strings.Contains(f.Path, "vendor/") ||
				strings.Contains(f.Path, "frontend/dist") {
				t.Errorf("excluded dir leaked a finding: %+v", f)
			}
		}
	})
}

// TestRenderMarkdown — the markdown report groups findings by file.
func TestRenderMarkdown(t *testing.T) {
	findings := []Finding{
		{Path: "internal/foo/a.go", Line: 3, Text: "// Hola mundo", Reason: "spanish-sentence"},
		{Path: "internal/foo/a.go", Line: 7, Text: "// Adiós mundo", Reason: "spanish-sentence"},
		{Path: "internal/bar/b.go", Line: 1, Text: "// Gracias", Reason: "spanish-sentence"},
	}
	md := RenderMarkdown(findings)
	if !strings.Contains(md, "internal/foo/a.go") {
		t.Errorf("markdown missing a.go: %s", md)
	}
	if !strings.Contains(md, "internal/bar/b.go") {
		t.Errorf("markdown missing b.go: %s", md)
	}
	if !strings.Contains(md, "Hola") {
		t.Errorf("markdown missing Hola: %s", md)
	}
}

// TestScan_InlineExemptionMarker — a line carrying an inline
// `// l10n: <reason>` marker is exempt from reporting. The marker
// lets a file declare an intentional non-English snippet
// (Unicode test fixture, illustrative i18n example, etc.) without
// rewriting the snippet.
func TestScan_InlineExemptionMarker(t *testing.T) {
	withTempRepo(t, func(root string) {
		writeFile(t, root, "internal/foo/exempt.go", `package foo

// Esta función se ignora porque lleva el marker. // l10n: illustrative
func Bar() {}
`)
		findings, err := Scan(root)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		for _, f := range findings {
			if strings.Contains(f.Path, "exempt.go") {
				t.Errorf("line with // l10n: marker must be exempt, got %+v", f)
			}
		}
	})
}

// TestScan_PrecedingLineExemptionMarker — a line carrying a Spanish
// sentence preceded by a `// l10n: <reason>` marker on its own
// line is also exempt. This is the form slice 2 uses for multi-line
// Spanish snippets.
func TestScan_PrecedingLineExemptionMarker(t *testing.T) {
	withTempRepo(t, func(root string) {
		writeFile(t, root, "internal/foo/multi.go", `package foo

// l10n: technical-unicode-fixture
// Esta función valida el usuario antes de continuar.
func Foo() {}
`)
		findings, err := Scan(root)
		if err != nil {
			t.Fatalf("Scan: %v", err)
		}
		for _, f := range findings {
			if strings.Contains(f.Path, "multi.go") && strings.Contains(f.Text, "Esta función") {
				t.Errorf("line after // l10n: marker must be exempt, got %+v", f)
			}
		}
	})
}

