package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Source-preserving parse/patch contract for Node-RED settings.js.
//
// Settings.js is executable JavaScript, not a JSON document. A
// model-and-regenerate approach can silently discard functions, require()
// expressions, middleware, third-party settings, comments and unknown keys.
// This file makes the active settings.js source authoritative and applies
// narrowly scoped managed edits without regenerating unrelated JavaScript.
//
// Ownership model
// ===============
//
//   - "Managed" keys are the entries NRCC edits through its UI. They are
//     declared in managedSettingKeys below and must match a catalog entry
//     in nodered_compatibility.go. Every other top-level entry in the
//     module.exports object literal is "unmanaged".
//
//   - The contract this module guarantees is byte-for-byte preservation of
//     every unmanaged region: keys not in the managed list, comments,
//     whitespace and content outside the module.exports object literal.
//
// Contract guarantees
// ===================
//
//   1. Managed scalar edits (ApplyScalarEdit) replace the matching top-level
//      `key: <value>,` line in place. The original indentation is preserved
//      when present.
//
//   2. Managed block edits (ApplyBlockEdit) replace the matching top-level
//      `key: { ... },` block in place. Nested objects, string literals,
//      template literals, line comments and block comments inside the block
//      are tokenised, not regex-matched (see findTopLevelBlock).
//
//   3. If the managed key is absent, the entry is appended just before the
//      closing brace of the module.exports object literal. The original
//      closing brace, newline and any trailing content are preserved.
//
//   4. ApplyScalarEdit / ApplyBlockEdit return ok=false when the source is
//      not a recognisable module.exports object literal. Callers must
//      surface the malformed source rather than coerce a partial result.
//
//   5. SourcePatch applies a batch of edits and rolls back atomically: if
//      any single edit fails, the original content is returned untouched.

// managedSettingKeys enumerates the top-level keys NRCC rewrites through its
// structured configuration UI. Tests use this list to assert preservation
// guarantees for unmanaged regions. Anything not on this list is unmanaged
// and must round-trip verbatim through every edit.
var managedSettingKeys = []string{
	"uiPort",
	"uiHost",
	"httpAdminRoot",
	"httpNodeRoot",
	"httpStatic",
	"flowFile",
	"flowFilePretty",
	"userDir",
	"nodesDir",
	"lang",
	"disableEditor",
	"projectsEnabled",
	"adminAuth",
	"httpNodeAuth",
	"httpStaticAuth",
	"editorTheme",
	"logging",
	"runtimeState",
	"credentialSecret",
	"functionGlobalContext",
	"env",
}

// IsManagedSettingKey reports whether key is one NRCC edits through its
// configuration UI. False means the key is operator-owned and must be
// preserved verbatim by any source-preserving patch.
func IsManagedSettingKey(key string) bool {
	for _, k := range managedSettingKeys {
		if k == key {
			return true
		}
	}
	return false
}

// ManagedSettingKeys returns a copy of the managed-key list. Tests and
// diagnostics consume this slice; callers must not mutate the returned slice
// to mutate the package state.
func ManagedSettingKeys() []string {
	out := make([]string, len(managedSettingKeys))
	copy(out, managedSettingKeys)
	return out
}

// ErrSourceNotExports is returned by the patch helpers when the active
// source is not a recognisable module.exports object literal. The contract
// demands that callers surface this rather than coerce a partial result.
var ErrSourceNotExports = errors.New("source is not a module.exports object literal")

// ApplyScalarEdit replaces the top-level `key: <scalar>` entry with `value`.
// Returns the patched source and ok=true on success. When ok is false the
// returned source equals the input and the caller must treat the patch as
// failed (see ErrSourceNotExports).
//
// The substitution preserves every byte that does not belong to the matched
// entry, including comments, whitespace, unmanaged keys and any content
// outside the module.exports object literal.
//
// Idempotency: when the existing scalar's textual representation already
// equals value, the line is left untouched. This makes "Save with the same
// value" a true no-op on disk and keeps the operator's exact spelling
// (e.g. `process.env.PORT || 1880`) intact instead of forcing the
// renderer to overwrite it with the canonical form.
func ApplyScalarEdit(content, key, value string) (string, bool) {
	start, end, ok := findTopLevelBlock(content, key)
	if !ok {
		// Not present — try to append before the closing brace of the
		// module.exports object. If we can't find a closing brace, the
		// source is not a recognisable module.exports object literal.
		idx, ok := findModuleExportsClosingBrace(content)
		if !ok {
			return content, false
		}
		// Detect the indentation of an existing line so the inserted entry
		// matches the file's style. Fall back to two spaces (Node-RED's
		// canonical settings.js style).
		indent := detectIndent(content)
		insertion := fmt.Sprintf("%s%s: %s,\n", indent, key, value)
		return content[:idx] + insertion + content[idx:], true
	}
	// Idempotency check: compare the existing scalar's textual form
	// against the new value. When they match, the line is already what
	// we would have written, so leave the source untouched.
	if existingScalarEquals(content[start:end], key, value) {
		return content, true
	}
	// Replace just the line content (from start to end) but preserve the
	// leading whitespace of that line so indentation is stable across edits.
	indent := content[start:leadingWhitespaceEnd(content, start)]
	replacement := fmt.Sprintf("%s%s: %s,", indent, key, value)
	return content[:start] + replacement + content[end:], true
}

// existingScalarEquals reports whether the rendered scalar at line [start,end)
// already equals `value`. The matched segment includes `key: <scalar>,`
// without the leading whitespace; we strip the key, the colon, the trailing
// comma and any internal whitespace, then compare against value.
func existingScalarEquals(segment, key, value string) bool {
	colon := strings.Index(segment, ":")
	if colon < 0 {
		return false
	}
	after := segment[colon+1:]
	// Strip leading whitespace.
	after = strings.TrimLeft(after, " \t")
	// Strip trailing comma and any trailing whitespace.
	after = strings.TrimRight(after, " \t")
	after = strings.TrimSuffix(after, ",")
	after = strings.TrimRight(after, " \t")
	return after == value
}

// ApplyBlockEdit replaces or appends a block entry (object/array literal).
// blockContent must already include its own leading indentation and trailing
// comma. The function reuses the brace-counting scanner so nested objects,
// string literals and comments inside the replaced block are tokenised
// rather than regex-matched.
func ApplyBlockEdit(content, key, blockContent string) (string, bool) {
	start, end, ok := findTopLevelBlock(content, key)
	if !ok {
		idx, ok := findModuleExportsClosingBrace(content)
		if !ok {
			return content, false
		}
		indent := detectIndent(content)
		insertion := "\n" + indent + strings.TrimLeft(blockContent, " \t")
		return content[:idx] + insertion + content[idx:], true
	}
	return content[:start] + blockContent + content[end:], true
}

// SourceEdit describes one managed edit to apply via SourcePatch.
type SourceEdit struct {
	Key      string // top-level setting key
	Value    string // for Scalar edits: the rendered value (e.g. "3000", "\"info\"")
	Block    string // for Block edits: the full replacement block including its leading indent and trailing comma
	IsBlock  bool   // when true, Value is ignored and Block is used
	Inserted bool   // Inserted is true after Apply if the edit appended a new entry rather than replacing an existing one
}

// SourcePatchResult is the outcome of a SourcePatch call.
type SourcePatchResult struct {
	Content  string
	Inserted []string // keys whose entries were appended because they were not present in the source
	Replaced []string // keys whose entries were replaced in place
}

// SourcePatch applies a sequence of managed edits to the source. Edits are
// processed in order. If any edit fails the original content is returned
// untouched and the failure is wrapped with ErrSourceNotExports.
func SourcePatch(content string, edits []SourceEdit) (SourcePatchResult, error) {
	current := content
	result := SourcePatchResult{Content: content}

	for _, edit := range edits {
		// Detect replacement vs insertion BEFORE the edit so the diagnostic
		// fields accurately describe what happened to the original source.
		_, _, existed := findTopLevelBlock(current, edit.Key)

		var patched string
		var ok bool
		if edit.IsBlock {
			patched, ok = ApplyBlockEdit(current, edit.Key, edit.Block)
		} else {
			patched, ok = ApplyScalarEdit(current, edit.Key, edit.Value)
		}
		if !ok {
			return SourcePatchResult{Content: content}, fmt.Errorf("apply %q: %w", edit.Key, ErrSourceNotExports)
		}
		current = patched
		if existed {
			result.Replaced = append(result.Replaced, edit.Key)
		} else {
			result.Inserted = append(result.Inserted, edit.Key)
		}
	}

	result.Content = current
	return result, nil
}

// findModuleExportsClosingBrace returns the byte index of the closing '}'
// of the outermost module.exports object literal on its own line, plus ok.
// The match must be the closing brace of the root object — deeper '}' are
// excluded because we want the trailing brace of `module.exports = { ... }`.
//
// The match is performed by scanning for a '}' that appears at column 0 (or
// preceded only by whitespace) on its own line. This is the canonical
// Node-RED settings.js shape.
func findModuleExportsClosingBrace(content string) (int, bool) {
	re := regexp.MustCompile(`(?m)^[ \t]*\}[ \t]*$`)
	idx := re.FindStringIndex(content)
	if idx == nil {
		return 0, false
	}
	return idx[0], true
}

// detectIndent returns the most common leading whitespace prefix on non-empty
// lines of content. Used to keep inserted entries stylistically consistent
// with the file. Returns two spaces when no consistent indent is detected.
func detectIndent(content string) string {
	counts := map[string]int{}
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		end := leadingWhitespaceEnd(line, 0)
		if end == 0 {
			continue
		}
		counts[line[:end]]++
	}
	best := ""
	bestN := 0
	for indent, n := range counts {
		if n > bestN {
			best = indent
			bestN = n
		}
	}
	if best == "" {
		return "  "
	}
	return best
}

// leadingWhitespaceEnd returns the index just past the leading whitespace of
// the line that starts at lineStart. When lineStart already points past the
// line's leading whitespace the function returns lineStart.
func leadingWhitespaceEnd(s string, lineStart int) int {
	i := lineStart
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}
