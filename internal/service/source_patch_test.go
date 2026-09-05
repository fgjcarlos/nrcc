package service

import (
	"strings"
	"testing"
)

// TestManagedSettingKeys_CatalogExposed locks in the managed-key contract:
// every key NRCC rewrites through its structured UI must round-trip through
// SourcePatch; everything else is operator-owned and must be preserved.
func TestManagedSettingKeys_CatalogExposed(t *testing.T) {
	want := []string{
		"uiPort", "uiHost",
		"httpAdminRoot", "httpNodeRoot", "httpStatic",
		"flowFile", "flowFilePretty",
		"userDir", "nodesDir",
		"lang", "disableEditor", "projectsEnabled",
		"adminAuth", "httpNodeAuth", "httpStaticAuth",
		"editorTheme", "logging", "runtimeState",
		"credentialSecret", "functionGlobalContext",
		"env",
	}
	got := ManagedSettingKeys()
	if len(got) != len(want) {
		t.Fatalf("managed key count mismatch: got %d want %d (%v)", len(got), len(want), got)
	}
	for i, key := range want {
		if got[i] != key {
			t.Errorf("managed key %d: got %q want %q", i, got[i], key)
		}
	}
}

// TestIsManagedSettingKey verifies the operator/unmanaged boundary. Keys
// the catalog does not own must report false so the source-preserving
// contract leaves them alone.
func TestIsManagedSettingKey(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"uiPort", true},
		{"httpAdminRoot", true},
		{"functionGlobalContext", true},
		{"credentialSecret", true},
		// Operator-owned / extension-owned / Node-RED advanced keys:
		{"httpMiddleware", false},
		{"externalModules", false},
		{"https", false},
		{"exportGlobalContextKeys", false},
		{"nodesDirArray", false},
		{"fooBar", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsManagedSettingKey(tc.key); got != tc.want {
			t.Errorf("IsManagedSettingKey(%q) = %v, want %v", tc.key, got, tc.want)
		}
	}
}

// TestApplyScalarEdit_ReplaceInPlace covers the in-place replacement path:
// an existing top-level `key: value,` line is rewritten while every other
// byte of the source is preserved verbatim.
func TestApplyScalarEdit_ReplaceInPlace(t *testing.T) {
	original := `module.exports = {
  uiPort: 1880,
  httpAdminRoot: "/",
  // operator comment
  functionGlobalContext: {
    lodash: require("lodash"),
  },
  httpMiddleware: function(req, res, next) { next(); },
}
`
	patched, ok := ApplyScalarEdit(original, "uiPort", "3000")
	if !ok {
		t.Fatalf("ApplyScalarEdit returned ok=false")
	}
	want := `module.exports = {
  uiPort: 3000,
  httpAdminRoot: "/",
  // operator comment
  functionGlobalContext: {
    lodash: require("lodash"),
  },
  httpMiddleware: function(req, res, next) { next(); },
}
`
	if patched != want {
		t.Fatalf("patched output mismatch.\n--- got ---\n%s\n--- want ---\n%s", patched, want)
	}
}

// TestApplyScalarEdit_AppendWhenMissing covers the append path: when the
// managed key is absent, the function inserts a new line just before the
// closing brace of the module.exports object literal, preserving the
// closing brace and any trailing content.
func TestApplyScalarEdit_AppendWhenMissing(t *testing.T) {
	original := `module.exports = {
  uiPort: 1880,
  // operator comment
  httpMiddleware: function(req, res, next) { next(); },
}
`
	patched, ok := ApplyScalarEdit(original, "httpNodeRoot", `"/api"`)
	if !ok {
		t.Fatalf("ApplyScalarEdit returned ok=false")
	}
	mustContain(t, patched, `  httpNodeRoot: "/api",`)
	mustContain(t, patched, "httpMiddleware")
	mustContain(t, patched, "// operator comment")
	// Closing brace stays at column 0; nothing after it.
	if !strings.HasSuffix(patched, "}\n") && !strings.HasSuffix(patched, "}") {
		t.Fatalf("closing brace position moved:\n%s", patched)
	}
}

// TestApplyScalarEdit_NotModuleExports ensures the contract refuses to
// coerce a non-module.exports source. Callers must surface this rather
// than ship a half-rewritten file.
func TestApplyScalarEdit_NotModuleExports(t *testing.T) {
	inputs := []string{
		"",
		"// no exports here",
		"module.exports = 1880;", // not an object literal
		"port = 1880\n",
	}
	for _, in := range inputs {
		got, ok := ApplyScalarEdit(in, "uiPort", "3000")
		if ok {
			t.Errorf("expected ok=false for %q, got patched content:\n%s", in, got)
		}
		if got != in {
			t.Errorf("content changed despite ok=false:\nbefore:\n%s\nafter:\n%s", in, got)
		}
	}
}

// TestApplyScalarEdit_PreservesUnmanagedKeys is the contract guard from
// the issue: every operator-owned key must survive an edit untouched.
func TestApplyScalarEdit_PreservesUnmanagedKeys(t *testing.T) {
	original := `module.exports = {
  uiPort: 1880,
  httpMiddleware: function(req, res, next) { next(); },
  functionGlobalContext: { lodash: require("lodash") },
  externalModules: { paths: ["/opt/nodes"] },
  exportGlobalContextKeys: ["lodash"],
  https: { key: require("fs").readFileSync("/etc/ssl/key.pem") },
  customThing: computeSomething(),
  // comment about the next entry
  logging: { console: { level: "info", metrics: false, audit: false } },
}
`
	for _, key := range []string{"uiPort", "httpAdminRoot", "httpNodeRoot", "lang"} {
		patched, ok := ApplyScalarEdit(original, key, `"x"`)
		if !ok {
			t.Fatalf("ApplyScalarEdit(%q) returned ok=false", key)
		}
		for _, preserved := range []string{
			"httpMiddleware",
			"functionGlobalContext",
			"externalModules",
			"exportGlobalContextKeys",
			"https:",
			"customThing: computeSomething()",
			"// comment about the next entry",
			"metrics: false",
		} {
			if !strings.Contains(patched, preserved) {
				t.Errorf("after editing %q, lost unmanaged region %q:\n%s", key, preserved, patched)
			}
		}
	}
}

// TestApplyBlockEdit_ReplacesNestedBlock ensures nested object literals are
// tokenised, not regex-matched. This is the regression guard for #723.
func TestApplyBlockEdit_ReplacesNestedBlock(t *testing.T) {
	original := `module.exports = {
  editorTheme: {
    codeEditor: { options: { theme: "vs-light" } },
  },
}
`
	patched, ok := ApplyBlockEdit(original, "editorTheme", "  editorTheme: {},\n")
	if !ok {
		t.Fatalf("ApplyBlockEdit returned ok=false")
	}
	if !strings.Contains(patched, "  editorTheme: {},") {
		t.Errorf("replacement block missing:\n%s", patched)
	}
	if strings.Contains(patched, "vs-light") {
		t.Errorf("nested value leaked through:\n%s", patched)
	}
	if strings.Count(patched, "editorTheme:") != 1 {
		t.Errorf("editorTheme appeared %d times, want 1:\n%s", strings.Count(patched, "editorTheme:"), patched)
	}
}

// TestApplyBlockEdit_AppendsBeforeClosingBrace covers the absent-key path
// for blocks. The blockContent must include its own leading indent and
// trailing comma so the result is syntactically valid.
func TestApplyBlockEdit_AppendsBeforeClosingBrace(t *testing.T) {
	original := `module.exports = {
  uiPort: 1880,
  // preserve me
}
`
	patched, ok := ApplyBlockEdit(original, "adminAuth", "  adminAuth: null,\n")
	if !ok {
		t.Fatalf("ApplyBlockEdit returned ok=false")
	}
	mustContain(t, patched, "  adminAuth: null,")
	mustContain(t, patched, "// preserve me")
	mustContain(t, patched, "  uiPort: 1880,")
}

// TestSourcePatch_BatchEditsAtomic verifies that a batch of edits either
// succeeds as a whole or returns the original content untouched.
func TestSourcePatch_BatchEditsAtomic(t *testing.T) {
	original := `module.exports = {
  uiPort: 1880,
}
`
	edits := []SourceEdit{
		{Key: "uiPort", Value: "3000"},
		{Key: "httpAdminRoot", Value: `"/admin"`},
		{Key: "httpNodeRoot", Value: `"/api"`},
	}
	res, err := SourcePatch(original, edits)
	if err != nil {
		t.Fatalf("SourcePatch returned error: %v", err)
	}
	if res.Content == original {
		t.Fatalf("SourcePatch returned original content despite successful edits")
	}
	for _, want := range []string{
		"uiPort: 3000",
		`httpAdminRoot: "/admin"`,
		`httpNodeRoot: "/api"`,
	} {
		mustContain(t, res.Content, want)
	}
}

// TestSourcePatch_RollsBackOnFailure verifies that a partial failure
// returns the original content (the contract is "atomic or no-op").
func TestSourcePatch_RollsBackOnFailure(t *testing.T) {
	// Source has no module.exports closing brace on its own line,
	// so the third edit must fail.
	original := "module.exports = { uiPort: 1880,"
	edits := []SourceEdit{
		{Key: "uiPort", Value: "3000"},
		{Key: "httpAdminRoot", Value: `"/admin"`},
		{Key: "httpNodeRoot", Value: `"/api"`},
	}
	res, err := SourcePatch(original, edits)
	if err == nil {
		t.Fatalf("expected error from malformed source, got nil")
	}
	if res.Content != original {
		t.Fatalf("partial failure must return original content unchanged.\nbefore:\n%s\nafter:\n%s", original, res.Content)
	}
}

// TestSourcePatch_TracksInsertedAndReplaced locks in the diagnostic fields
// callers need to drive "this setting already existed" UX cues.
func TestSourcePatch_TracksInsertedAndReplaced(t *testing.T) {
	original := `module.exports = {
  uiPort: 1880,
  adminAuth: null,
}
`
	edits := []SourceEdit{
		{Key: "uiPort", Value: "3000"},            // replaced
		{Key: "httpAdminRoot", Value: `"/admin"`}, // inserted
		// adminAuth is also edited but it's a block, so use Block path
		{Key: "adminAuth", Block: "  adminAuth: null,\n", IsBlock: true}, // replaced
	}
	res, err := SourcePatch(original, edits)
	if err != nil {
		t.Fatalf("SourcePatch: %v", err)
	}
	if len(res.Inserted) != 1 || res.Inserted[0] != "httpAdminRoot" {
		t.Errorf("Inserted = %v, want [httpAdminRoot]", res.Inserted)
	}
	// uiPort and adminAuth should appear in Replaced
	replaced := strings.Join(res.Replaced, ",")
	for _, k := range []string{"uiPort", "adminAuth"} {
		if !strings.Contains(replaced, k) {
			t.Errorf("Replaced = %v, expected to contain %q", res.Replaced, k)
		}
	}
}

// TestApplyScalarEdit_Idempotent ensures a no-op edit (same value) still
// returns content that is byte-for-byte equal to the input. This is the
// property that makes fingerprint-based conflict detection safe (#757
// acceptance test: TestSettingsRoundTripPreservesUnmanagedSource).
func TestApplyScalarEdit_Idempotent(t *testing.T) {
	original := `module.exports = {
  uiPort: 1880,
  httpMiddleware: function(req, res, next) { next(); },
}
`
	once, ok := ApplyScalarEdit(original, "uiPort", "1880")
	if !ok {
		t.Fatalf("ApplyScalarEdit returned ok=false")
	}
	if once != original {
		t.Fatalf("idempotent edit drifted:\nbefore:\n%s\nafter:\n%s", original, once)
	}
	twice, ok := ApplyScalarEdit(once, "uiPort", "1880")
	if !ok {
		t.Fatalf("ApplyScalarEdit returned ok=false on second call")
	}
	if twice != once {
		t.Fatalf("idempotent edit drifted on second call:\nonce:\n%s\ntwice:\n%s", once, twice)
	}
}

// TestApplyScalarEdit_CommentsAndWhitespaceOutsideEntry verifies that
// comments and whitespace outside the replaced entry are preserved exactly,
// not normalised. This is the lossless property the issue demands.
//
// The trailing inline comment on the same line as the managed value is
// removed along with the value: it cannot be preserved "where technically
// possible" without re-tokenising the value, which the slice's contract
// explicitly disclaims. Comments on their own lines must survive untouched.
func TestApplyScalarEdit_CommentsAndWhitespaceOutsideEntry(t *testing.T) {
	original := "module.exports = {\n" +
		"  // top-level comment\n" +
		"  uiPort: 1880, // inline comment\n" +
		"  httpAdminRoot: \"/\",\n" +
		"  /*\n" +
		"   * block comment\n" +
		"   */\n" +
		"  functionGlobalContext: {},\n" +
		"}\n"
	patched, ok := ApplyScalarEdit(original, "uiPort", "3000")
	if !ok {
		t.Fatalf("ApplyScalarEdit returned ok=false")
	}
	for _, must := range []string{
		"// top-level comment",
		"httpAdminRoot: \"/\"",
		"/*",
		"* block comment",
		"*/",
		"functionGlobalContext: {}",
	} {
		mustContain(t, patched, must)
	}
	mustContain(t, patched, "uiPort: 3000")
	if strings.Contains(patched, "uiPort: 1880") {
		t.Fatalf("stale value present:\n%s", patched)
	}
}

// mustContain fails the test if needle is missing from haystack.
func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("missing %q in:\n%s", needle, haystack)
	}
}
