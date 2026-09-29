package service

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCatalogDocSync verifies that
// docs/handbook/configuration/setting-catalog.md mirrors the
// `nodeRED5Catalog` slice one-to-one.
//
// The test is deliberately strict: any drift between the catalog
// slice (the single source of truth) and the documented table
// fails CI. Adding a new entry therefore requires updating both the
// Go struct and the Markdown table in the same commit (see the
// "Adding a new entry" section of setting-catalog.md).
//
// Run with: go test ./internal/service/ -run TestCatalogDocSync
func TestCatalogDocSync(t *testing.T) {
	docPath := catalogDocPath(t)
	entries := NodeRED5Catalog()
	if len(entries) == 0 {
		t.Fatal("NodeRED5Catalog() returned no entries; refusing to scan an empty catalog")
	}

	rows, err := parseCatalogTable(docPath)
	if err != nil {
		t.Fatalf("parse %s: %v", docPath, err)
	}

	if got, want := len(rows), len(entries); got != want {
		t.Errorf("catalog table row count = %d, want %d (one per nodeRED5Catalog entry). Update both files in the same commit.", got, want)
	}

	// Index the documented rows by Key for set-equality and per-field
	// assertions. The order of entries in the markdown table does not
	// have to match the Go slice order.
	byKey := make(map[string]catalogRow, len(rows))
	for _, r := range rows {
		if _, dup := byKey[r.key]; dup {
			t.Errorf("duplicate key %q in %s", r.key, docPath)
			continue
		}
		byKey[r.key] = r
	}

	// Set-equality of keys. Reject any key present in one side but not
	// the other; rejection must come before per-field comparisons so
	// the failure message is actionable.
	for _, e := range entries {
		row, ok := byKey[e.Key]
		if !ok {
			t.Errorf("catalog entry %q is documented in Go but missing from %s", e.Key, docPath)
			continue
		}
		assertCatalogRow(t, e, row)
	}
	for _, r := range rows {
		found := false
		for _, e := range entries {
			if e.Key == r.key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("catalog row %q in %s has no matching nodeRED5Catalog entry", r.key, docPath)
		}
	}

	// Pinned catalog version: the markdown header must mention the
	// same "5.0.6" (or whatever nodeRED5CatalogVersion holds). This
	// catches the operator-visible "we changed the catalog version
	// without bumping the docs" mistake.
	raw, err := os.ReadFile(docPath) // #nosec G304 -- docPath is derived from runtime.Caller(0)+filepath.Join("docs/handbook/..."), not from user input.
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}
	needle := "Catalog version is pinned by `nodeRED5CatalogVersion = \"" + nodeRED5CatalogVersion + "\"`"
	if !strings.Contains(string(raw), needle) {
		t.Errorf("%s is missing the pinned-version note: %q", docPath, needle)
	}
}

// catalogRow is the parsed view of one catalog table row.
type catalogRow struct {
	key             string
	shape           string
	def             string
	validation      string
	secret          bool
	restartRequired bool
	uiEditable      bool
}

func assertCatalogRow(t *testing.T, e SettingCatalogEntry, r catalogRow) {
	t.Helper()
	if e.Shape != r.shape {
		t.Errorf("%s.Shape: catalog=%q doc=%q", e.Key, e.Shape, r.shape)
	}
	if e.Default != r.def {
		t.Errorf("%s.Default: catalog=%q doc=%q", e.Key, e.Default, r.def)
	}
	if e.Validation != r.validation {
		t.Errorf("%s.Validation: catalog=%q doc=%q", e.Key, e.Validation, r.validation)
	}
	if e.Secret != r.secret {
		t.Errorf("%s.Secret: catalog=%v doc=%v", e.Key, e.Secret, r.secret)
	}
	if e.RestartRequired != r.restartRequired {
		t.Errorf("%s.RestartRequired: catalog=%v doc=%v", e.Key, e.RestartRequired, r.restartRequired)
	}
	if e.UIEditable != r.uiEditable {
		t.Errorf("%s.UIEditable: catalog=%v doc=%v", e.Key, e.UIEditable, r.uiEditable)
	}
}

// parseCatalogTable returns the rows of the catalog table from
// docs/handbook/configuration/setting-catalog.md. The parser is
// intentionally narrow: it expects the header
//
//	| Key | Shape | Default | Validation | Secret | RestartRequired | UIEditable |
//
// followed by a markdown separator line and then `| <key> | ...
// | <✓| > | <✓| > | <✓| > |` rows in any order.
//
// The parser tolerates inline pipes that are escaped with a backslash
// (e.g. `process.env.PORT \| 1880`); splitMarkdownRow consumes `\|` as
// a literal `|` before splitting the row on raw `|`.
func parseCatalogTable(path string) ([]catalogRow, error) {
	f, err := os.Open(path) // #nosec G304 -- path is derived from runtime.Caller(0)+filepath.Join("docs/handbook/..."), not from user input.
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	const wantHeader = "| Key | Shape | Default | Validation | Secret | RestartRequired | UIEditable |"

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	foundHeader := false
	foundSeparator := false
	var rows []catalogRow
	for scanner.Scan() {
		line := scanner.Text()
		if !foundHeader {
			if strings.TrimSpace(line) == wantHeader {
				foundHeader = true
			}
			continue
		}
		if !foundSeparator {
			// markdown table separator, e.g. |---|---|...|
			if strings.HasPrefix(strings.TrimSpace(line), "|") && strings.Contains(line, "---") {
				foundSeparator = true
			}
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "|") {
			// Table ended.
			break
		}
		cells := splitMarkdownRow(trimmed)
		if len(cells) != 7 {
			return nil, fmt.Errorf("line %q: expected 7 cells, got %d", line, len(cells))
		}
		key := strings.TrimSpace(cells[0])
		row := catalogRow{
			key:        key,
			shape:      strings.TrimSpace(cells[1]),
			def:        strings.TrimSpace(cells[2]),
			validation: strings.TrimSpace(cells[3]),
		}
		var err error
		if row.secret, err = parseCheckCell(cells[4]); err != nil {
			return nil, fmt.Errorf("%s secret marker: %w", key, err)
		}
		if row.restartRequired, err = parseCheckCell(cells[5]); err != nil {
			return nil, fmt.Errorf("%s restart marker: %w", key, err)
		}
		if row.uiEditable, err = parseCheckCell(cells[6]); err != nil {
			return nil, fmt.Errorf("%s ui-editable marker: %w", key, err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !foundHeader {
		return nil, fmt.Errorf("header %q not found in %s", wantHeader, path)
	}
	if !foundSeparator {
		return nil, fmt.Errorf("markdown table separator not found after header in %s", path)
	}
	return rows, nil
}

// splitMarkdownRow splits a leading-and-trailing-pipe markdown row
// into its cells. It tolerates optional leading and trailing pipes
// and treats `\|` as an escaped pipe that does NOT split a cell.
// Any trailing escape backslash is left as a literal because the
// catalog has no such cells today.
func splitMarkdownRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	var cells []string
	var current strings.Builder
	i := 0
	for i < len(line) {
		c := line[i]
		if c == '\\' && i+1 < len(line) && line[i+1] == '|' {
			current.WriteByte('|')
			i += 2
			continue
		}
		if c == '|' {
			cells = append(cells, current.String())
			current.Reset()
			i++
			continue
		}
		current.WriteByte(c)
		i++
	}
	cells = append(cells, current.String())
	return cells
}

// parseCheckCell interprets a check / cross cell as a boolean.
// Accepts ✓, ✔, yes, true (any case) and rejects ✗, ✘, no, false,
// empty. Anything else is reported as an error so a typo in the
// doc fails the test instead of silently flipping a flag.
func parseCheckCell(cell string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(cell)) {
	case "✓", "✔", "yes", "true":
		return true, nil
	case "", "✗", "✘", "no", "false":
		return false, nil
	}
	return false, fmt.Errorf("unrecognized marker %q (allowed: ✓ ✔ ✗ ✘ yes no true false, case-insensitive)", cell)
}

// catalogDocPath resolves docs/handbook/configuration/setting-catalog.md
// relative to this test file's location. Using runtime.Caller keeps
// the test working when the package is built from any cwd.
func catalogDocPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed; cannot locate repo root")
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	return filepath.Join(repoRoot, "docs", "handbook", "configuration", "setting-catalog.md")
}
