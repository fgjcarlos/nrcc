package service

import (
	"fmt"
	"strings"
	"testing"

	"nrcc/internal/model"
)

type fakeRunner struct {
	outputs map[string]string
	errors  map[string]error
}

func (f fakeRunner) LookPath(name string) (string, error) {
	key := "lookpath " + name
	if err, ok := f.errors[key]; ok {
		return "", err
	}
	if output, ok := f.outputs[key]; ok {
		return output, nil
	}
	return "/usr/bin/" + name, nil
}

func (f fakeRunner) Run(_ string, name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	if err, ok := f.errors[key]; ok {
		return "", err
	}
	if output, ok := f.outputs[key]; ok {
		return output, nil
	}
	return "", fmt.Errorf("unexpected command: %s", key)
}

func TestValidatePackageName(t *testing.T) {
	t.Parallel()

	valid := []string{"lodash", "@scope/pkg", "node-red-contrib-test", "pkg.name"}
	for _, pkg := range valid {
		if _, err := validatePackageName(pkg); err != nil {
			t.Fatalf("validatePackageName(%q) error = %v", pkg, err)
		}
	}

	invalid := []string{"", "pkg name", "../pkg", "file:../pkg", "git+https://x", "--save", "UPPER"}
	for _, pkg := range invalid {
		if _, err := validatePackageName(pkg); err == nil {
			t.Fatalf("validatePackageName(%q) error = nil", pkg)
		}
	}
}

func TestParseLibraryList(t *testing.T) {
	t.Parallel()

	items, err := parseLibraryList(`{"dependencies":{"@scope/pkg":{"version":"2.0.0"},"lodash":{"version":"4.17.21"}}}`)
	if err != nil {
		t.Fatalf("parseLibraryList() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("parseLibraryList() len = %d, want 2", len(items))
	}
	if items[0].Name != "@scope/pkg" || items[1].Name != "lodash" {
		t.Fatalf("parseLibraryList() items = %+v", items)
	}
}

func TestLibraryServiceInstallAndUninstall(t *testing.T) {
	t.Parallel()

	service := LibraryService{
		dataDir: t.TempDir(),
		runner: fakeRunner{
			outputs: map[string]string{
				"npm install lodash":      "installed",
				"npm uninstall lodash":    "removed",
				"npm ls --json --depth=0": `{"dependencies":{"lodash":{"version":"4.17.21"}}}`,
			},
			errors: map[string]error{},
		},
	}

	result, err := service.Install("lodash")
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if result.Package != (model.LibraryPackage{Name: "lodash", Version: "4.17.21", Direct: true}) {
		t.Fatalf("Install() package = %+v", result.Package)
	}

	removed, err := service.Uninstall("lodash")
	if err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if removed.Package.Name != "lodash" || removed.Operation != "uninstall" {
		t.Fatalf("Uninstall() result = %+v", removed)
	}
}

func TestOperationLock(t *testing.T) {
	t.Parallel()

	lock := NewOperationLock()
	release, err := lock.Acquire("installing", "lodash")
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	status := lock.Status()
	if !status.Busy || status.Type != "installing" {
		t.Fatalf("Status() = %+v", status)
	}

	if _, err := lock.Acquire("restoring", "backup"); err == nil {
		t.Fatal("Acquire() second call error = nil")
	}

	release()

	status = lock.Status()
	if status.Busy {
		t.Fatalf("Status() after release = %+v", status)
	}
}
