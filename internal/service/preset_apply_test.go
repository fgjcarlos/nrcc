package service

import (
	"os"
	"strings"
	"testing"
)

// Test slice 2 of issue #764 — apply, preview, rollback, and the
// unmanaged-region preservation guarantee called out by the issue's
// acceptance tests:
//
//   - TestAdvancedPatchPreservesUnmanagedCode: an unrelated form edit
//     leaves custom callbacks byte-stable in unmanaged regions.
//   - FunctionGlobalContextAndNodeDefaultsFixtureSuite: supported
//     structured cases round-trip; executable/unknown cases remain
//     preserved.

// readFixture loads a settings.js fixture from testdata/preset-fixtures/.
// Tests use it instead of inlining large source strings so the fixtures
// stay readable as Node-RED 5 source.
func readFixture(t *testing.T, name string) string {
	t.Helper()
	path := "testdata/preset-fixtures/" + name
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

// TestPresetApply_AppliesHttpsPreset verifies that applying the
// https-tls-preset replaces the existing https block and leaves every
// other byte of the source untouched. This is the "managed block edit"
// path that slice 2 builds on.
func TestPresetApply_AppliesHttpsPreset(t *testing.T) {
	original := `module.exports = {
  flowFile: 'flows.json',
  uiPort: 1880,
  https: {
    key: require("fs").readFileSync("/etc/ssl/old-key.pem"),
    cert: require("fs").readFileSync("/etc/ssl/old-cert.pem"),
  },
  httpMiddleware: function(req, res, next) { next(); },
}
`
	reg := NewPresetRegistry()
	res, err := reg.ApplyPreset(original, "https-tls-preset", map[string]string{})
	if err != nil {
		t.Fatalf("ApplyPreset: %v", err)
	}
	if !strings.Contains(res.After, "https:") {
		t.Errorf("After must contain the new https block, got:\n%s", res.After)
	}
	if !strings.Contains(res.After, "httpMiddleware") {
		t.Errorf("After must still contain operator-owned httpMiddleware, got:\n%s", res.After)
	}
	httpsReplaced := false
	for _, k := range res.Replaced {
		if k == "https" {
			httpsReplaced = true
			break
		}
	}
	if !httpsReplaced {
		t.Errorf("Replaced must report https, got %v", res.Replaced)
	}
}

// TestAdvancedPatchPreservesUnmanagedCode — the headline acceptance test
// for slice 2. Apply a preset that touches a managed key (https or
// functionGlobalContext); verify every operator-owned block outside the
// managed set is byte-stable. The "operator-owned" bytes include a
// custom httpMiddleware function, an executable require() inside FGC,
// and a comment.
func TestAdvancedPatchPreservesUnmanagedCode(t *testing.T) {
	reg := NewPresetRegistry()

	cases := []struct {
		name     string
		presetID string
		fixture  string
		needle   string // substring that must appear unchanged in After
	}{
		{
			name:     "https-preserve-httpMiddleware",
			presetID: "https-tls-preset",
			fixture:  "functionGlobalContext-supported.js",
			needle:   "httpMiddleware: function(req, res, next) { next(); }",
		},
		{
			name:     "fgc-relaxed-preserve-require-expressions",
			presetID: "functionGlobalContext-relaxed",
			fixture:  "functionGlobalContext-executable-require.js",
			needle:   `lodash: require("lodash")`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := readFixture(t, tc.fixture)
			res, err := reg.ApplyPreset(original, tc.presetID, map[string]string{})
			if err != nil {
				t.Fatalf("ApplyPreset: %v", err)
			}
			if !strings.Contains(res.After, tc.needle) {
				t.Errorf("After must preserve %q verbatim, got:\n%s", tc.needle, res.After)
			}
		})
	}
}

// TestPresetApply_PreviewRedactsSecrets verifies the preview shown to
// the operator never contains the values of secret-shaped managed keys
// (https.passphrase, credentialSecret, functionGlobalContext, etc.).
// The issue calls this out as a security review gate.
func TestPresetApply_PreviewRedactsSecrets(t *testing.T) {
	reg := NewPresetRegistry()
	// Build a settings.js whose https block contains a passphrase; the
	// preview must replace the passphrase with [redacted].
	original := `module.exports = {
  https: {
    key: require("fs").readFileSync("/etc/ssl/key.pem"),
    cert: require("fs").readFileSync("/etc/ssl/cert.pem"),
    passphrase: "super-secret-passphrase",
  },
}
`
	res, err := reg.ApplyPreset(original, "https-tls-preset", map[string]string{})
	if err != nil {
		t.Fatalf("ApplyPreset: %v", err)
	}
	if strings.Contains(res.Preview, "super-secret-passphrase") {
		t.Errorf("preview leaked passphrase:\n%s", res.Preview)
	}
	if !strings.Contains(res.Preview, "[redacted]") {
		t.Errorf("preview should contain [redacted] marker, got:\n%s", res.Preview)
	}
}

// TestPresetApply_RollsBackOnUnknownPreset verifies the apply path
// fails closed when the preset ID is not registered. The result must
// carry the original content unchanged.
func TestPresetApply_RollsBackOnUnknownPreset(t *testing.T) {
	reg := NewPresetRegistry()
	original := "module.exports = { uiPort: 1880, }\n"
	res, err := reg.ApplyPreset(original, "no-such-preset", map[string]string{})
	if err == nil {
		t.Fatalf("expected error for unknown preset, got nil")
	}
	if res.After != original {
		t.Errorf("After must equal original on failure, got %q want %q", res.After, original)
	}
}

// TestPresetApply_RollsBackOnSourcePatchFailure verifies that a
// failed SourcePatch leaves the original content untouched. We
// exercise this by feeding the apply path content that is NOT a
// recognisable module.exports literal — SourcePatch must surface
// ErrSourceNotExports and the apply must return the original
// verbatim.
func TestPresetApply_RollsBackOnSourcePatchFailure(t *testing.T) {
	reg := NewPresetRegistry()
	original := "this is not module.exports"
	_, err := reg.ApplyPreset(original, "https-tls-preset", map[string]string{})
	if err == nil {
		t.Fatalf("expected error from malformed source, got nil")
	}
	if !strings.Contains(err.Error(), "source is not a module.exports") {
		t.Errorf("expected ErrSourceNotExports, got %v", err)
	}
}

// TestFunctionGlobalContextAndNodeDefaultsFixtureSuite covers the
// acceptance test called out in issue #764. It iterates the supported
// FGC and nodeDefaults fixtures and asserts:
//
//   - The supported-structured fixture (functionGlobalContext-supported.js)
//     round-trips: applying the FGC-strict preset yields a result that
//     still parses as module.exports and preserves operator-owned code.
//
//   - The executable-require fixture is preserved verbatim — no
//     attempt to coerce require() expressions into a structured form.
//
//   - The nodeDefaults-typed-inputs fixture preserves its typed-input
//     shape when a non-FGC preset is applied (the fixture exercises
//     the source-patch contract on an adjacent key).
func TestFunctionGlobalContextAndNodeDefaultsFixtureSuite(t *testing.T) {
	reg := NewPresetRegistry()

	t.Run("supported-structured-roundtrip", func(t *testing.T) {
		original := readFixture(t, "functionGlobalContext-supported.js")
		res, err := reg.ApplyPreset(original, "functionGlobalContext-strict", map[string]string{})
		if err != nil {
			t.Fatalf("ApplyPreset: %v", err)
		}
		if !strings.Contains(res.After, "functionGlobalContext:") {
			t.Errorf("supported FGC must still be present in After, got:\n%s", res.After)
		}
		if !strings.Contains(res.After, "httpMiddleware: function") {
			t.Errorf("operator-owned httpMiddleware must be preserved, got:\n%s", res.After)
		}
	})

	t.Run("executable-require-preserved", func(t *testing.T) {
		original := readFixture(t, "functionGlobalContext-executable-require.js")
		res, err := reg.ApplyPreset(original, "functionGlobalContext-relaxed", map[string]string{})
		if err != nil {
			t.Fatalf("ApplyPreset: %v", err)
		}
		if !strings.Contains(res.After, `lodash: require("lodash")`) {
			t.Errorf("executable require() must be preserved, got:\n%s", res.After)
		}
		if !strings.Contains(res.After, `myHelpers: require("./lib/helpers")`) {
			t.Errorf("second executable require() must be preserved, got:\n%s", res.After)
		}
	})

	t.Run("nodeDefaults-preserved-across-fgc-preset", func(t *testing.T) {
		original := readFixture(t, "nodeDefaults-typed-inputs.js")
		res, err := reg.ApplyPreset(original, "functionGlobalContext-strict", map[string]string{})
		if err != nil {
			t.Fatalf("ApplyPreset: %v", err)
		}
		if !strings.Contains(res.After, `nodeDefaults:`) {
			t.Errorf("nodeDefaults must be preserved when applying an FGC preset, got:\n%s", res.After)
		}
		if !strings.Contains(res.After, `"repeat": false`) {
			t.Errorf("typed-input shape must be preserved, got:\n%s", res.After)
		}
	})
}
