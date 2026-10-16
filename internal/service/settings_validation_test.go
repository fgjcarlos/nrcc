package service

import (
	"errors"
	"testing"
)

func TestSettingsInspectionDistinguishesValidationOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		want          error
	}{
		{"missing auth", "module.exports = { uiPort: 1880 };", ErrAdminAuthMissing},
		{"invalid syntax", "module.exports = { uiPort: 1880, broken: };", ErrSandboxSyntax},
		{"syntax after require", "require('os'); module.exports = { broken: };", ErrSandboxSyntax},
		{"runtime", "throw new Error('Expected SyntaxError');", ErrSandboxRuntime},
		{"undefined value", "module.exports = { uiPort: unknownPort };", ErrSandboxRuntime},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseAuthenticationSurfacesViaSandbox(tc.content)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if errors.Is(err, ErrSandboxHostAPI) || (tc.want != ErrSandboxSyntax && errors.Is(err, ErrSandboxSyntax)) {
				t.Fatalf("inspection error misclassified: %v", err)
			}
		})
	}
}

func TestSettingsInspectionBlocksHostAPIsExplicitly(t *testing.T) {
	for _, name := range []string{
		"require", "process", "Buffer", "globalThis", "global", "setTimeout", "setInterval", "fetch",
		"this.process", "Function('return this')().require", "this.Buffer", "this.fetch",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseAuthenticationSurfacesViaSandbox("var host = " + name + "; module.exports = {};")
			if !errors.Is(err, ErrSandboxHostAPI) || !errors.Is(err, ErrSandboxRuntime) {
				t.Fatalf("host API did not remain blocked with typed runtime error: %v", err)
			}
		})
	}
}
