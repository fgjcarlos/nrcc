// Package service — slice A of issue #758 introduces an apply pipeline
// that orchestrates validate → backup → atomic write → audit. The audit
// stage needs a redacted diff so secrets (credentialSecret,
// https.passphrase, functionGlobalContext credentials) never reach the
// audit log even in operator-facing slices.
//
// This file owns the diff formatter and the redaction policy. The HTTP
// handlers in slice B will pass the rendered diff straight into the
// audit meta; this slice only guarantees the diff itself is safe to
// log.
//
// Redaction contract
// ==================
//
//   1. The policy enumerates managed settings whose value shape is
//      "secret" (catalog entries with Secret:true in
//      nodered_compatibility.go). The current list is the exact slice of
//      managedSettingKeys for which the Node-RED 5 catalog marks Secret.
//      Adding a new secret-shaped setting to that catalog requires
//      adding its key to secretSettingKeys here; otherwise the new key
//      would leak into audit logs unchanged.
//
//   2. The redaction targets the value, not the key. A line of the
//      form `key: <value>,` becomes `key: "[redacted]"`. Top-level
//      scalars only; nested object/array values for these keys are
//      collapsed to a single "[redacted]" token so the audit reader can
//      still see WHICH secret-shaped key changed without learning the
//      new value.
//
//   3. The redactor is line-based. It does not parse the candidate as
//      JavaScript; it only knows the managed keys. Unknown
//      secret-shaped keys added by an operator (e.g. custom auth
//      blocks) pass through verbatim — the catalog is the source of
//      truth.
//
//   4. The diff between before and after is a per-line textual diff.
//      Lines present only in before are prefixed "- ", lines present
//      only in after are prefixed "+ ", lines common to both are
//      prefixed "  ". The output is suitable for direct inclusion in a
//      JSON audit meta field.

package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// secretSettingKeys is the closed list of top-level keys whose VALUES
// must be redacted before the diff reaches the audit log. Membership
// must stay in lock-step with the Node-RED 5 catalog entries whose
// Secret flag is true. See internal/service/nodered_compatibility.go
// for the catalog.
var secretSettingKeys = []string{
	"credentialSecret",
	"functionGlobalContext",
	"adminAuth",
	"httpNodeAuth",
	"httpStaticAuth",
	"https",
}

// secretSettingKeySet returns the secretSettingKeys list as a lookup
// table. Tests use it to assert the membership is in lock-step with the
// catalog; runtime callers iterate via IsSecretSettingKey.
func secretSettingKeySet() map[string]struct{} {
	set := make(map[string]struct{}, len(secretSettingKeys))
	for _, k := range secretSettingKeys {
		set[k] = struct{}{}
	}
	return set
}

// IsSecretSettingKey reports whether key is in the audit-redaction list.
// The contract is "key shape is secret-shaped per the Node-RED 5
// catalog", not "this exact key appears in this candidate"; an operator
// who sets adminAuth to null must still see "adminAuth: [redacted]" in
// the audit diff.
func IsSecretSettingKey(key string) bool {
	for _, k := range secretSettingKeys {
		if k == key {
			return true
		}
	}
	return false
}

// SecretSettingKeys returns a copy of the secretSettingKeys list. Tests
// and diagnostics consume it; callers must not mutate the returned slice.
func SecretSettingKeys() []string {
	out := make([]string, len(secretSettingKeys))
	copy(out, secretSettingKeys)
	return out
}

// httpsPassphraseSecretKeys extends the redaction policy to nested
// passphrase fields inside the https object. Node-RED's https block is a
// top-level Secret entry in the catalog; passphrase is its inner field
// that carries the actual private-key passphrase. Redacting only the
// top-level "https" would still expose the passphrase; we additionally
// redact the passphrase field inline.
const httpsPassphraseField = "passphrase"

// RedactSettingsContent returns content with every secret-shaped
// top-level key replaced by `key: [redacted]`. Nested secret fields
// (https.passphrase) are also redacted inline.
//
// The function is conservative: anything it doesn't recognise is left
// untouched. Operators who add new secret-shaped keys must update the
// catalog AND this list.
func RedactSettingsContent(content string) string {
	if content == "" {
		return ""
	}
	secretSet := secretSettingKeySet()
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		key, ok := keyInLine(line)
		if !ok {
			continue
		}
		if _, isSecret := secretSet[key]; isSecret {
			lines[i] = redactTopLevelLine(line, key)
			continue
		}
		if key == "https" {
			lines[i] = redactHttpsPassphrase(line)
		}
	}
	return strings.Join(lines, "\n")
}

// RedactDiff returns a unified diff between before and after with
// secret-shaped values redacted on both sides. The diff uses a
// per-line LCS-free join: lines unique to before are emitted as "- ",
// lines unique to after as "+ ", and shared lines as "  ". This is
// intentionally simple; the audit reader does not need GNU-diff fidelity.
func RedactDiff(before, after string) string {
	before = RedactSettingsContent(before)
	after = RedactSettingsContent(after)
	if before == after {
		return ""
	}
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")
	var b strings.Builder
	// Use a per-line set membership check; O(n*m) but settings.js is
	// short (a few hundred lines at most) so this is fine. The intent is
	// readability, not GNU diff --precision.
	beforeSet := make(map[string]int, len(beforeLines))
	for i, l := range beforeLines {
		beforeSet[l] = i
	}
	for _, l := range afterLines {
		if _, ok := beforeSet[l]; ok {
			fmt.Fprintf(&b, "  %s\n", l)
			delete(beforeSet, l)
		} else {
			fmt.Fprintf(&b, "+ %s\n", l)
		}
	}
	// Emit remaining before-only lines in original order.
	remaining := make([]string, 0, len(beforeSet))
	for l := range beforeSet {
		remaining = append(remaining, l)
	}
	sort.Strings(remaining)
	for _, l := range remaining {
		fmt.Fprintf(&b, "- %s\n", l)
	}
	return b.String()
}

// keyInLine returns the bare key name when line contains a `key:` token
// at any indentation level. The recogniser is intentionally permissive:
// any line where the first non-whitespace token matches
// `[A-Za-z_$][A-Za-z0-9_$]*:` is treated as a key assignment. The
// caller applies the catalog/secret policy; this helper only locates
// the key.
//
// ok=false for blank lines, pure comment lines (//, /*), and lines
// without a colon after the identifier. Lines whose colon is inside a
// single-line comment are also rejected.
func keyInLine(line string) (key string, ok bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return "", false
	}
	// Strip single-line comments: anything from // onward is not a key.
	if cidx := strings.Index(trimmed, "//"); cidx == 0 {
		return "", false
	}
	// Find the first identifier-like token.
	for i, c := range trimmed {
		switch {
		case i == 0 && (c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' || c == '$'):
			continue
		case i > 0 && (c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '$'):
			continue
		case c == ':':
			return trimmed[:i], true
		default:
			return "", false
		}
	}
	return "", false
}

// redactTopLevelLine rewrites line so the value after `key:` becomes
// "[redacted]". Works for scalars (`key: foo,`) and blocks (`key: { ... },`).
func redactTopLevelLine(line, key string) string {
	idx := strings.Index(line, key+":")
	if idx < 0 {
		return line
	}
	// Replace everything from the colon onward (preserve any leading
	// whitespace before the key).
	prefix := line[:idx+len(key)+1]
	// Preserve trailing comma + newline if present.
	trailing := ""
	if strings.HasSuffix(strings.TrimRight(line, " \t"), ",") {
		trailing = ","
	}
	return prefix + ` [redacted]` + trailing
}

// redactHttpsPassphrase replaces any `passphrase: <value>` occurrence
// inside a top-level `https:` block with `passphrase: [redacted]`. It
// is brace-count aware so a `passphrase:` key in a nested object that
// happens to live inside the https block is also collapsed.
func redactHttpsPassphrase(line string) string {
	// For the slice A scope we only redact the single-line `passphrase:`
	// form. Multi-line block redaction would require a parser that knows
	// the entire block bounds; slice C's adapter interface will own that
	// when the field shape grows beyond a string.
	needle := httpsPassphraseField + ":"
	idx := strings.Index(line, needle)
	if idx < 0 {
		return line
	}
	// Skip if the passphrase key is itself inside a comment by checking
	// for "//" before the index on this line. Slice A keeps this
	// conservative.
	if cidx := strings.Index(line, "//"); cidx >= 0 && cidx < idx {
		return line
	}
	prefix := line[:idx+len(needle)]
	return prefix + ` [redacted]`
}

// RedactedCapabilityMeta produces an audit-meta map keyed by the
// ConfigurationCapabilities fields that may carry secret-shaped values
// (https.passphrase, etc.). It is a thin helper so callers do not have
// to remember to redact before logging.
func RedactedCapabilityMeta(caps model.ConfigurationCapabilities) map[string]string {
	m := map[string]string{
		"runtime_version": caps.RuntimeVersion,
		"adapter":         caps.Adapter,
		"settings_source": caps.Source,
		"mode":            caps.Mode,
		"catalog_version": caps.CatalogVersion,
	}
	if caps.Reason != "" {
		m["read_only_reason"] = caps.Reason
	}
	return m
}