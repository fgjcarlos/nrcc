package service

import (
	"errors"
	"testing"
)

func TestValidatePackageNameAcceptsValid(t *testing.T) {
	valid := []string{
		"node-red-dashboard",
		"@scope/package",
		"@flowfuse/node-red-dashboard",
		"node-red-contrib-mqtt@2.1.0",
		"@scope/pkg@^1.2.3",
		"my-package",
		"pkg123",
		"a",
	}

	for _, name := range valid {
		t.Run(name, func(t *testing.T) {
			if err := ValidatePackageName(name); err != nil {
				t.Fatalf("expected %q to be valid, got: %v", name, err)
			}
		})
	}
}

func TestValidatePackageNameRejectsInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"dot-dot traversal", "../../x"},
		{"absolute path", "/tmp/pkg"},
		{"relative path", "./local-pkg"},
		{"https URL", "https://evil.com/pkg.tgz"},
		{"git URL", "git+ssh://github.com/x/y"},
		{"whitespace command", "pkg && rm -rf /"},
		{"semicolon injection", "pkg;whoami"},
		{"pipe injection", "pkg|cat /etc/passwd"},
		{"backtick injection", "pkg`id`"},
		{"dollar injection", "pkg$(whoami)"},
		{"tab whitespace", "pkg\there"},
		{"newline", "pkg\nmalicious"},
		{"uppercase start", "MyPackage"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePackageName(tc.input)
			if err == nil {
				t.Fatalf("expected %q to be rejected, got nil", tc.input)
			}
			if !errors.Is(err, ErrInvalidPackageName) {
				t.Fatalf("expected ErrInvalidPackageName, got: %v", err)
			}
		})
	}
}
