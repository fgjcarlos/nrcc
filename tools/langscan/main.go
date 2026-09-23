// Package langscan implements the language-policy scanner for issue #768.
//
// The scanner walks a repository rooted at a given directory and emits
// a list of suspected non-English artefacts in non-frontend files.
// Spanish sentences in user-visible UI strings are out of scope — they
// migrate to i18n catalogs under issue #767 — so the scanner must
// exclude frontend/src and only flag comments / dev messages / docs
// / scripts / CI workflows.
//
// Detection heuristic (intentionally conservative for slice 1):
//   - A line is flagged when it contains a Spanish-specific character
//     (á, é, í, ó, ú, ñ, ¿, ¡, ü) AND has two or more words AND is
//     not entirely composed of legitimate technical Unicode
//     (math symbols, section signs, arrows, Greek letters).
//   - The exclusion list (frontend/src, frontend/dist, node_modules,
//     vendor, .git, the langscan tool itself) keeps the report
//     narrow enough to be actionable.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Severity of a finding. The slice 1 scanner emits only "warn"; future
// rules can add "info" or "error" without breaking the contract.
type Severity string

const (
	SeverityWarn Severity = "warn"
)

// Finding is one suspected non-English artefact.
type Finding struct {
	Path     string   `json:"path"`
	Line     int      `json:"line"`
	Column   int      `json:"column"`
	Text     string   `json:"text"`
	Reason   string   `json:"reason"`
	Severity Severity `json:"severity"`
}

// excludedDirs are walked-through but never reported.
var excludedDirs = []string{
	"node_modules",
	"vendor",
	"frontend/dist",
	".git",
	".next",
	"dist",
	"build",
	"tmp",
	"tools/langscan/testdata",
}

// excludedPathPrefixes — entire path prefixes that are out of scope
// for issue #768 (UI strings deferred to issue #767 i18n catalogs).
var excludedPathPrefixes = []string{
	"frontend/src",   // user-visible UI strings — handled by #767 i18n catalogs
	"frontend/e2e",   // e2e specs mirror UI strings; migrated with the catalog in #767
	"auditoria",      // historical audit docs are immutable; only NEW docs are English
	"tools/langscan", // self-reference: the scanner's own docs/tests use Spanish fixtures
	"odd/tasks",      // ODD workspace docs may quote non-English artefacts verbatim as evidence
	".agents/skills", // third-party agent skills (installed by the agent-skills framework)
}

// scannedExtensions limits the walk to known technical files. Locale
// files (frontend/src/locales/*) are inside an excluded prefix and
// therefore naturally skipped.
var scannedExtensions = map[string]bool{
	".go":   true,
	".ts":   true,
	".tsx":  true,
	".js":   true,
	".jsx":  true,
	".md":   true,
	".yml":  true,
	".yaml": true,
	".sh":   true,
	".bash": true,
	".toml": true,
	".json": true,
}

// spanishChars are the character set that distinguishes Spanish text
// from English ASCII.
const spanishChars = "áéíóúñüÁÉÍÓÚÑÜ¿¡"

// isTechnicalOnly returns true when the only non-ASCII characters in
// line are from a known technical Unicode subset (math, section
// signs, Greek, arrows, box-drawing). Such lines are legitimate code
// or comments and must not be flagged.
func isTechnicalOnly(line string) bool {
	for _, r := range line {
		if r <= 127 {
			continue
		}
		switch {
		case unicode.Is(unicode.Mn, r), // combining marks (rare in plain text)
			unicode.Is(unicode.Sk, r):
			// modifier/symbol letters — include
		case unicode.IsPunct(r):
			// punctuation — include
		case unicode.IsSymbol(r):
			// math/section/etc — INCLUDE; these are legitimate technical chars
		case unicode.IsSpace(r):
			// space separators — include
		default:
			return false
		}
	}
	return true
}

// hasSpanishChar reports whether line contains at least one
// Spanish-specific character.
func hasSpanishChar(line string) bool {
	return strings.ContainsAny(line, spanishChars)
}

// wordCount returns the number of whitespace-separated tokens.
func wordCount(line string) int {
	return len(strings.Fields(line))
}

// isExcluded reports whether path starts with any of the excluded
// prefixes.
func isExcluded(path string) bool {
	for _, prefix := range excludedPathPrefixes {
		if strings.HasPrefix(path, prefix+"/") || path == prefix {
			return true
		}
	}
	return false
}

// hasExcludedDir reports whether any directory in path is in the
// excluded-dirs list.
func hasExcludedDir(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, p := range parts {
		for _, e := range excludedDirs {
			if p == e {
				return true
			}
		}
	}
	return false
}

// exemptionMarker exempts the line (or the line that follows it)
// from the language-policy scan. Contributors attach it inline
// (`... // l10n: <reason>`) or on its own line above the snippet
// they want to exempt. The scanner keeps both forms so a contributor
// can choose whichever fits the artefact.
const exemptionMarker = "// l10n:"

// hasExemptionMarker reports whether line carries an exemption marker.
func hasExemptionMarker(line string) bool {
	return strings.Contains(line, exemptionMarker)
}

// lineShouldFlag returns the reason string if line should be reported,
// empty string otherwise. A line that carries or follows an exemption
// marker is always skipped so contributors can opt out for
// intentional non-English snippets (Unicode fixtures, illustrative
// i18n examples, etc.).
func lineShouldFlag(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	if hasExemptionMarker(trimmed) {
		return ""
	}
	if !hasSpanishChar(trimmed) {
		return ""
	}
	if wordCount(trimmed) < 2 {
		return ""
	}
	if isTechnicalOnly(trimmed) {
		return ""
	}
	return "spanish-sentence"
}

// Scan walks root and returns findings.
func Scan(root string) ([]Finding, error) {
	var findings []Finding

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if hasExcludedDir(rel) {
				return fs.SkipDir
			}
			return nil
		}
		if isExcluded(rel) {
			return nil
		}
		if hasExcludedDir(filepath.Dir(rel)) {
			return nil
		}
		ext := filepath.Ext(rel)
		if !scannedExtensions[ext] {
			return nil
		}
		fileFindings, err := scanFile(root, rel)
		if err != nil {
			return fmt.Errorf("scan %s: %w", rel, err)
		}
		findings = append(findings, fileFindings...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Line < findings[j].Line
	})
	return findings, nil
}

// scanFile reads path under root and returns findings for each
// Spanish-sentence line. A line is skipped when it carries an
// exemption marker (// l10n: <reason>) inline OR when the previous
// line is exactly a marker line.
func scanFile(root, rel string) ([]Finding, error) {
	// #nosec G304 G703 — root comes from CLI arg, rel comes from filepath.WalkDir over root.
	full := filepath.Join(root, rel)
	// #nosec G304 — full is rooted at the user-supplied repo root and filtered by exclusions upstream.
	f, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var findings []Finding
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 4*1024*1024) // long lines
	prevWasMarker := false
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		text := scanner.Text()
		if prevWasMarker {
			prevWasMarker = false
			continue
		}
		if hasExemptionMarker(strings.TrimSpace(text)) {
			prevWasMarker = true
			continue
		}
		reason := lineShouldFlag(text)
		if reason == "" {
			continue
		}
		col := strings.IndexAny(text, spanishChars)
		if col < 0 {
			col = 0
		}
		findings = append(findings, Finding{
			Path:     rel,
			Line:     lineNo,
			Column:   col,
			Text:     strings.TrimRight(text, " \t"),
			Reason:   reason,
			Severity: SeverityWarn,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return findings, nil
}

// RenderMarkdown renders findings as a human-readable report grouped
// by file. The output is suitable for inclusion in odd/tasks/.
func RenderMarkdown(findings []Finding) string {
	if len(findings) == 0 {
		return "# Language policy scan\n\nNo findings.\n"
	}
	byFile := map[string][]Finding{}
	for _, f := range findings {
		byFile[f.Path] = append(byFile[f.Path], f)
	}
	var paths []string
	for p := range byFile {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var b strings.Builder
	fmt.Fprintf(&b, "# Language policy scan\n\n")
	fmt.Fprintf(&b, "Total findings: **%d** across **%d** files.\n\n", len(findings), len(paths))
	for _, p := range paths {
		fmt.Fprintf(&b, "## `%s`\n\n", p)
		for _, f := range byFile[p] {
			fmt.Fprintf(&b, "- L%d: %s — `%s`\n", f.Line, f.Reason, f.Text)
		}
		fmt.Fprintf(&b, "\n")
	}
	return b.String()
}

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: langscan <root>")
		os.Exit(2)
	}
	root, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve root: %v\n", err)
		os.Exit(2)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		fmt.Fprintln(os.Stderr, "root must be an existing directory")
		os.Exit(2)
	}
	findings, err := Scan(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan error:", err)
		os.Exit(2)
	}
	format := flag.String("format", "json", "output format: json or markdown")
	if *format == "markdown" {
		fmt.Print(RenderMarkdown(findings))
		return
	}
	if len(findings) == 0 {
		fmt.Println("[]")
		return
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(findings); err != nil {
		fmt.Fprintln(os.Stderr, "encode error:", err)
		os.Exit(2)
	}
}
