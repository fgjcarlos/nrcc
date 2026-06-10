package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/composedof2/nrcc/internal/model"
)

type fakeSystemd struct{}

func (fakeSystemd) IsAvailable() bool                       { return true }
func (fakeSystemd) DaemonReload() error                     { return nil }
func (fakeSystemd) EnableAndStart(string) error             { return nil }
func (fakeSystemd) Stop(string) error                       { return nil }
func (fakeSystemd) Disable(string) error                    { return nil }
func (fakeSystemd) GetServiceStatus(string) (string, error) { return "inactive", nil }

func seedInstalledLayout(t *testing.T) model.InstallLayout {
	t.Helper()
	dir := t.TempDir()
	layout := model.InstallLayout{
		SystemdUnit: filepath.Join(dir, "nrcc.service"),
		BinaryPath:  filepath.Join(dir, "nrcc"),
		ConfigDir:   filepath.Join(dir, "etc"),
		DataDir:     filepath.Join(dir, "data"),
		ServiceName: "nrcc",
		ServiceUser: "nrcc",
	}
	if err := os.WriteFile(layout.SystemdUnit, []byte("unit"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.BinaryPath, []byte("bin"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layout.ConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layout.DataDir, 0755); err != nil {
		t.Fatal(err)
	}
	return layout
}

// TestUninstall_PurgeRemovesUserGroupAndData is the #283 regression: a full
// uninstall must remove the data directory AND the nrcc system user/group.
func TestUninstall_PurgeRemovesUserGroupAndData(t *testing.T) {
	layout := seedInstalledLayout(t)
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	withMockInstallerExec(t, recordFile, "", "")
	withMockInstallerUserLookup(t, true, true) // user and group exist

	svc := &InstallerService{systemd: fakeSystemd{}, layout: layout}
	if err := svc.Uninstall(context.Background(), model.UninstallOpts{Purge: true, SkipPrompt: true}); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if _, err := os.Stat(layout.DataDir); !os.IsNotExist(err) {
		t.Error("data directory should be removed on purge")
	}

	data, _ := os.ReadFile(recordFile)
	cmds := string(data)
	if !strings.Contains(cmds, "userdel") {
		t.Errorf("expected userdel to be called; commands:\n%s", cmds)
	}
	if !strings.Contains(cmds, "groupdel") {
		t.Errorf("expected groupdel to be called; commands:\n%s", cmds)
	}
}

// TestUninstall_KeepDataPreservesDataAndUser ensures --keep-data keeps both the
// data directory and the system user (for a later reinstall).
func TestUninstall_KeepDataPreservesDataAndUser(t *testing.T) {
	layout := seedInstalledLayout(t)
	recordFile := filepath.Join(t.TempDir(), "commands.log")
	withMockInstallerExec(t, recordFile, "", "")
	withMockInstallerUserLookup(t, true, true)

	svc := &InstallerService{systemd: fakeSystemd{}, layout: layout}
	if err := svc.Uninstall(context.Background(), model.UninstallOpts{KeepData: true, SkipPrompt: true}); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if _, err := os.Stat(layout.DataDir); err != nil {
		t.Errorf("data directory should be kept with --keep-data: %v", err)
	}

	data, _ := os.ReadFile(recordFile)
	if strings.Contains(string(data), "userdel") {
		t.Errorf("userdel must not run with --keep-data; commands:\n%s", string(data))
	}
}
