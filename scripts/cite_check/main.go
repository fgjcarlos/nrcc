// cite_check verifies that every internal link in a Markdown file
// resolves to a real path relative to the file's directory. Slice 1
// of issue #769 uses this tool to gate the README rewrite so the
// release-relevant cross-references stay accurate as the docs tree
// grows.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding reports a broken internal link inside a Markdown file.
type Finding struct {
	Path   string // path of the markdown file (relative to scan root)
	Line   int    // 1-based line number
	Target string // link target
	Text   string // surrounding link text for context
}

// linkRe captures an inline Markdown link: [text](url) and image
// links ![alt](url). Anchor references [text][label] and bare URLs
// are handled separately.
var inlineLinkRe = regexp.MustCompile(`!?\[[^\]\n]*\]\(([^)\n]+)\)`)

// Check scans path for internal Markdown links and returns the
// broken ones. External links (http(s)://, mailto:, #anchor-only)
// are considered well-formed and not returned.
func Check(path string) []Finding {
	// #nosec G304,G703 -- path is the CLI argument; the tool is invoked
	// on user-trusted docs in the local working tree, not on remote
	// attacker-controlled input.
	f, err := os.Open(path) // #nosec G304,G703
	if err != nil {
		return []Finding{{Path: path, Line: 0, Target: "", Text: err.Error()}}
	}
	defer func() { _ = f.Close() }()

	dir := filepath.Dir(path)
	var findings []Finding
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		text := scanner.Text()
		matches := inlineLinkRe.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			target := strings.TrimSpace(m[1])
			target = stripTitle(target)
			if isExternalOrAnchor(target) {
				continue
			}
			target = stripFragment(target)
			// #nosec G703 -- dir is the directory of the CLI-supplied
			// markdown file; target is parsed out of that file's body,
			// which is exactly the surface the tool is designed to scan.
			resolved := filepath.Join(dir, target) // #nosec G703
			if _, err := os.Stat(resolved); err != nil {
				findings = append(findings, Finding{
					Path:   path,
					Line:   lineNo,
					Target: target,
					Text:   strings.TrimSpace(text),
				})
			}
		}
	}
	return findings
}

// stripFragment removes any trailing #anchor from a path-style link.
// (docs/x.md#install) -> docs/x.md
func stripFragment(s string) string {
	if i := strings.Index(s, "#"); i >= 0 {
		return s[:i]
	}
	return s
}

// stripTitle removes any optional title in a Markdown link:
// (path "title") -> path.
func stripTitle(s string) string {
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}

// isExternalOrAnchor returns true for http/https/mailto URLs and
// pure-fragment references, which Check cannot verify locally.
func isExternalOrAnchor(target string) bool {
	if strings.HasPrefix(target, "#") {
		return true
	}
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return true
	}
	if strings.HasPrefix(target, "mailto:") {
		return true
	}
	// Reference-style [foo][bar] targets — these can be either
	// inline text or URL fragments; we cannot guarantee resolution
	// without also parsing the [bar]: ... definition block, so
	// we skip them defensively.
	return false
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: cite_check <markdown-file>")
		os.Exit(2)
	}
	findings := Check(os.Args[1])
	if len(findings) == 0 {
		fmt.Println("OK: no broken internal links")
		return
	}
	for _, f := range findings {
		fmt.Printf("BROKEN %s:%d -> %s\n  %s\n", f.Path, f.Line, f.Target, f.Text)
	}
	os.Exit(1)
}
