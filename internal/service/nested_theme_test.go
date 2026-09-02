package service

import (
	"os"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TestSave_NestedEditorThemeBlock_ReplacesInPlace covers the regression
// from #723: when settings.js already had an editorTheme block with
// 2-level nested content (codeEditor.options.theme), the regex-based
// replaceBlockKey failed to match, and the new block was appended
// alongside the old one. Node-RED then kept showing the old values
// (and on strict parsers, the file became invalid JS).
func TestSave_NestedEditorThemeBlock_ReplacesInPlace(t *testing.T) {
	dir := t.TempDir()
	svc := NewIsolatedConfigService(dir)
	settingsPath := dir + "/settings.js"

	existing := `module.exports = {
  uiPort: 1880,
  httpAdminRoot: "/",
  httpNodeRoot: "/",
  editorTheme: {
    page: { title: "Old Title" },
    header: { title: "Old Header" },
    codeEditor: { lib: "monaco", options: { theme: "vs-light" } },
  },
  logging: {
    console: { level: 'info', metrics: false, audit: false },
  },
}
`
	if err := os.WriteFile(settingsPath, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	cfg := model.DefaultNodeRedConfig()
	cfg.EditorTheme = map[string]any{
		"page":   map[string]any{"title": "New Title"},
		"header": map[string]any{"title": "New Header"},
	}
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	after := readFileString(t, settingsPath)
	t.Logf("Saved settings.js:\n%s", after)

	if strings.Count(after, "editorTheme:") != 1 {
		t.Fatalf("editorTheme appears %d times, want 1:\n%s", strings.Count(after, "editorTheme:"), after)
	}
	if !strings.Contains(after, "title: \"New Title\"") {
		t.Fatalf("expected 'New Title' in output:\n%s", after)
	}
	if !strings.Contains(after, "title: \"New Header\"") {
		t.Fatalf("expected 'New Header' in output:\n%s", after)
	}
	if strings.Contains(after, "Old Title") || strings.Contains(after, "Old Header") || strings.Contains(after, "vs-light") {
		t.Fatalf("stale values leaked through:\n%s", after)
	}
	if !strings.Contains(after, "logging:") {
		t.Fatalf("logging block was dropped:\n%s", after)
	}
}

// TestReplaceBlockKey_DeeplyNested_NestingNoLongerBreaks covers the
// arbitrary-nesting property of the brace-counting implementation. The
// previous regex-based replaceBlockKey could not match an existing block
// that contained a 2-level nested object (e.g.
// `codeEditor: { lib, options: { theme } }`) and fell through to
// "append a duplicate". Each case here is checked by content presence
// rather than byte-exact equality — minor whitespace drift in the
// append path is acceptable; duplicate keys are not.
func TestReplaceBlockKey_DeeplyNested_NestingNoLongerBreaks(t *testing.T) {
	cases := []struct {
		name           string
		content        string
		key            string
		block          string
		mustContain    []string
		mustNotContain []string
		count          map[string]int // key -> exact occurrences expected
	}{
		{
			name: "two levels (regression from #723)",
			content: `m = {
  editorTheme: {
    codeEditor: { options: { theme: "vs-light" } },
  },
}
`,
			key:   "editorTheme",
			block: "  editorTheme: {},\n",
			mustContain: []string{
				"  editorTheme: {},",
			},
			mustNotContain: []string{
				"vs-light",
				"codeEditor",
				"options: { theme:",
			},
			count: map[string]int{"editorTheme:": 1},
		},
		{
			name: "three levels",
			content: `m = {
  foo: { a: { b: { c: { d: 1 } } } },
  other: "keep",
}
`,
			key:   "foo",
			block: "  foo: {},\n",
			mustContain: []string{
				"  foo: {},",
				`other: "keep"`,
			},
			mustNotContain: []string{
				"a: { b:",
				"d: 1",
			},
			count: map[string]int{"foo:": 1},
		},
		{
			name: "block with object literal containing braces in a string",
			content: `m = {
  theme: { page: { css: "body { color: red; }" } },
}
`,
			key:   "theme",
			block: "  theme: {},\n",
			mustContain: []string{
				"  theme: {},",
			},
			mustNotContain: []string{
				`css: "body { color: red; }"`,
			},
			count: map[string]int{"theme:": 1},
		},
		{
			name: "scalar adminAuth: null",
			content: `m = {
  adminAuth: null,
  logging: {},
}
`,
			key:   "adminAuth",
			block: "  adminAuth: {},\n",
			mustContain: []string{
				"  adminAuth: {},",
			},
			mustNotContain: []string{
				"adminAuth: null",
			},
			count: map[string]int{"adminAuth:": 1},
		},
		{
			name: "missing key gets appended before closing brace",
			content: `m = {
  uiPort: 1880,
}
`,
			key:   "editorTheme",
			block: "  editorTheme: {},\n",
			mustContain: []string{
				"  uiPort: 1880,",
				"  editorTheme: {},",
			},
			count: map[string]int{"editorTheme:": 1},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := replaceBlockKey(tc.content, tc.key, tc.block)
			for _, want := range tc.mustContain {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q in output:\n%s", want, got)
				}
			}
			for _, banned := range tc.mustNotContain {
				if strings.Contains(got, banned) {
					t.Errorf("found banned %q in output:\n%s", banned, got)
				}
			}
			for key, expected := range tc.count {
				if n := strings.Count(got, key); n != expected {
					t.Errorf("key %q appears %d times, want %d:\n%s", key, n, expected, got)
				}
			}
		})
	}
}
