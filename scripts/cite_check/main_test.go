package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Test slice 1 of issue #769 — the README rewrite must keep all
// internal links resolveable. The cite_check tool guards that
// invariant by parsing Markdown links out of a file and verifying
// relative file targets exist relative to the doc's directory.

func TestCheck_RejectsBrokenInternalLink(t *testing.T) {
	root := withTempDir(t)
	mustWriteFile(t, filepath.Join(root, "README.md"), "# title\n\nsee [guide](docs/guide.md)\n")
	mustWriteFile(t, filepath.Join(root, "docs", "guide.md"), "# guide\n")
	// Now break the link by removing the target.
	mustRemove(t, filepath.Join(root, "docs", "guide.md"))

	findings := Check(filepath.Join(root, "README.md"))
	if len(findings) == 0 {
		t.Fatalf("expected at least one broken-link finding, got none")
	}
	if findings[0].Target != "docs/guide.md" {
		t.Errorf("expected first finding to target docs/guide.md, got %s", findings[0].Target)
	}
}

func TestCheck_AcceptsResolvedInternalLink(t *testing.T) {
	root := withTempDir(t)
	mustWriteFile(t, filepath.Join(root, "README.md"), "# title\n\nsee [guide](docs/guide.md)\n")
	mustWriteFile(t, filepath.Join(root, "docs", "guide.md"), "# guide\n")

	findings := Check(filepath.Join(root, "README.md"))
	if len(findings) != 0 {
		t.Errorf("expected zero findings, got %d: %+v", len(findings), findings)
	}
}

func TestCheck_AcceptsExternalLinks(t *testing.T) {
	root := withTempDir(t)
	mustWriteFile(t, filepath.Join(root, "README.md"), "# title\n\nsee [gh](https://github.com/foo/bar) and [issue](#1)\n")

	findings := Check(filepath.Join(root, "README.md"))
	if len(findings) != 0 {
		t.Errorf("external links and anchors should not be flagged, got %+v", findings)
	}
}

func TestCheck_AcceptsSiblingDirectoryRelativeLink(t *testing.T) {
	root := withTempDir(t)
	mustWriteFile(t, filepath.Join(root, "README.md"), "# title\n\nsee [CONTRIBUTING](../CONTRIBUTING.md)\n")
	mustWriteFile(t, filepath.Join(root, "CONTRIBUTING.md"), "# contributing\n")

	// Change to root's parent to simulate repo-root invocation
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, "README.md"), filepath.Join(root, "sub", "README.md")); err != nil {
		t.Fatal(err)
	}

	findings := Check(filepath.Join(root, "sub", "README.md"))
	// ../ resolves to root where CONTRIBUTING.md exists
	if len(findings) != 0 {
		t.Errorf("expected zero findings, got %+v", findings)
	}
}

func TestCheck_PreservesInAnchorLinks(t *testing.T) {
	root := withTempDir(t)
	mustWriteFile(t, filepath.Join(root, "README.md"), "# title\n\nsee [anchor](docs/guide.md#install) and [bad](docs/guide.md#missing)\n")
	mustWriteFile(t, filepath.Join(root, "docs", "guide.md"), "# guide\n## install\n")

	findings := Check(filepath.Join(root, "README.md"))
	// Anchors aren't checked (too brittle for markdown auto-anchors),
	// but the file target must exist.
	if len(findings) != 0 {
		t.Errorf("anchor link should not fail when the file exists, got %+v", findings)
	}
}

func TestCheck_HandlesParenthesesInLinkText(t *testing.T) {
	root := withTempDir(t)
	mustWriteFile(t, filepath.Join(root, "README.md"), "# title\n\nsee [aria (a11y)](docs/a11y.md)\n")
	mustWriteFile(t, filepath.Join(root, "docs", "a11y.md"), "# aria\n")

	findings := Check(filepath.Join(root, "README.md"))
	if len(findings) != 0 {
		t.Errorf("parens in link text should parse cleanly, got %+v", findings)
	}
}

func TestCheck_IgnoresReferenceStyleLinks(t *testing.T) {
	root := withTempDir(t)
	// Reference-style links [foo][bar] should be best-effort parsed
	// when the definition is present, ignored when absent (no
	// file-resolution guarantee).
	mustWriteFile(t, filepath.Join(root, "README.md"), "# title\n\nfoo [bar] not present.\n")
	// Also handle image-style: ![alt](image.png)
	mustWriteFile(t, filepath.Join(root, "image.png"), "fake png")

	findings := Check(filepath.Join(root, "README.md"))
	// Reference style without definition must not be flagged.
	for _, f := range findings {
		if f.Target == "" {
			t.Errorf("reference-style link should not produce an empty-target finding")
		}
	}
}

func TestCheck_ReportsLineNumber(t *testing.T) {
	root := withTempDir(t)
	mustWriteFile(t, filepath.Join(root, "README.md"), "# title\n\n## Quick start\n\nsee [missing](docs/missing.md)\n")
	findings := Check(filepath.Join(root, "README.md"))
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Line != 5 {
		t.Errorf("expected line 5, got %d", findings[0].Line)
	}
}

func withTempDir(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	return d
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustRemove(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}
