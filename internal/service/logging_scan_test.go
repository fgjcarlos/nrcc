package service

import (
	"strings"
	"testing"
)

// TestScanJSObjectEntries_FindsDirectChildren asserts the scanner returns
// the direct children of an object literal.
func TestScanJSObjectEntries_FindsDirectChildren(t *testing.T) {
	body := " level: 'info', metrics: false, audit: false "

	entries := scanJSObjectEntries(body)

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.key)
	}
	want := []string{"level", "metrics", "audit"}
	if len(names) != len(want) {
		t.Fatalf("scanJSObjectEntries returned %v, want %v", names, want)
	}
	for i, name := range want {
		if names[i] != name {
			t.Errorf("entry %d = %q, want %q", i, names[i], name)
		}
	}
	if got := body[entries[0].valueStart:entries[0].valueEnd]; got != "'info'" {
		t.Errorf("the level value = %q, want 'info'", got)
	}
	if got := body[entries[1].valueStart:entries[1].valueEnd]; got != "false" {
		t.Errorf("the metrics value = %q, want false", got)
	}
}

// TestScanJSObjectEntries_IgnoresNestedProperties asserts a property that
// repeats deeper in the tree is not mistaken for a sibling.
func TestScanJSObjectEntries_IgnoresNestedProperties(t *testing.T) {
	body := " handler: function (options) { return { level: 'nested' }; }, level: 'info' "

	entries := scanJSObjectEntries(body)

	if len(entries) != 2 {
		t.Fatalf("scanJSObjectEntries returned %d entries, want 2", len(entries))
	}
	if entries[0].key != "handler" {
		t.Errorf("the first entry = %q, want handler", entries[0].key)
	}
	if entries[1].key != "level" {
		t.Errorf("the second entry = %q, want level", entries[1].key)
	}
	if got := body[entries[1].valueStart:entries[1].valueEnd]; got != "'info'" {
		t.Errorf("the level value = %q, want 'info'", got)
	}
}

// TestScanJSObjectEntries_SkipsLiteralsAndComments asserts a string, a
// template literal and a comment never break the scan.
func TestScanJSObjectEntries_SkipsLiteralsAndComments(t *testing.T) {
	body := " note: 'a, b: c', /* level: 'commented' */ level: 'info', template: `x, y: ${z}` "

	entries := scanJSObjectEntries(body)

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.key)
	}
	want := []string{"note", "level", "template"}
	if len(names) != len(want) {
		t.Fatalf("scanJSObjectEntries returned %v, want %v", names, want)
	}
	for i, name := range want {
		if names[i] != name {
			t.Errorf("entry %d = %q, want %q", i, names[i], name)
		}
	}
}

// TestScanJSObjectEntries_ReadsQuotedKeys asserts a quoted property name
// is read as the property it declares.
func TestScanJSObjectEntries_ReadsQuotedKeys(t *testing.T) {
	entries := scanJSObjectEntries(" 'level': 'info' ")

	if len(entries) != 1 {
		t.Fatalf("scanJSObjectEntries returned %d entries, want 1", len(entries))
	}
	if entries[0].key != "level" {
		t.Errorf("the entry key = %q, want level", entries[0].key)
	}
}

// TestInsertJSEntries_KeepsTheLayout asserts the insertion matches the
// layout of the object literal it lands in.
func TestInsertJSEntries_KeepsTheLayout(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		literal string
		want    string
	}{
		{
			name:    "multi-line body",
			body:    "\n    console: { level: 'info' },\n  ",
			literal: "internal: { level: 'warn' }",
			want:    "\n    console: { level: 'info' },\n    internal: { level: 'warn' },\n  ",
		},
		{
			name:    "single-line body",
			body:    " console: { level: 'info' } ",
			literal: "internal: { level: 'warn' }",
			want:    " console: { level: 'info' }, internal: { level: 'warn' } ",
		},
		{
			name:    "empty body",
			body:    "",
			literal: "console: { level: 'info' }",
			want:    " console: { level: 'info' } ",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entries := scanJSObjectEntries(tc.body)
			text, start, end, ok := insertJSEntries(tc.body, entries, []string{tc.literal})
			if !ok {
				t.Fatal("insertJSEntries reported no insertion")
			}
			got := tc.body[:start] + text + tc.body[end:]
			if got != tc.want {
				t.Errorf("insertJSEntries produced %q, want %q", got, tc.want)
			}
		})
	}
}

// TestJSBlockBody_ReadsTheBlockBody asserts the helper reads the body of
// the whole entry of a block, not only of a bare literal.
func TestJSBlockBody_ReadsTheBlockBody(t *testing.T) {
	body, ok := jsBlockBody("  logging: {\n    console: { level: 'info' },\n  },")
	if !ok {
		t.Fatal("jsBlockBody found no object literal")
	}
	if !strings.Contains(body, "console") {
		t.Errorf("jsBlockBody returned %q, want the console entry", body)
	}

	if _, ok := jsBlockBody("logging: null"); ok {
		t.Error("jsBlockBody must not accept a scalar value")
	}
}
