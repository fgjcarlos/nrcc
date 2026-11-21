package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// assertReplacedByRename proves a writer published path through a new inode
// (temp file + rename) instead of truncating the live file in place: a hard
// link taken before the write must still hold the previous bytes (#1016).
func assertReplacedByRename(t *testing.T, path, previous string, write func()) {
	t.Helper()
	link := path + ".before"
	if err := os.Link(path, link); err != nil {
		t.Fatalf("link %s: %v", path, err)
	}
	write()
	got, err := os.ReadFile(link)
	if err != nil {
		t.Fatalf("read link: %v", err)
	}
	if string(got) != previous {
		t.Fatalf("%s was rewritten in place; previous inode now holds %q", filepath.Base(path), got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("%s mode = %o, want 600", filepath.Base(path), info.Mode().Perm())
	}
}

func TestWriteDotenvReplacesFileAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	writeTestFile(t, path, "A=1\n")
	assertReplacedByRename(t, path, "A=1\n", func() {
		if err := WriteDotenv(dir, "A=2\n"); err != nil {
			t.Fatalf("WriteDotenv: %v", err)
		}
	})
	if got, _ := ReadDotenv(dir); got != "A=2\n" {
		t.Fatalf("ReadDotenv = %q, want %q", got, "A=2\n")
	}
}

func TestBackupSaveConfigReplacesFileAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, backupConfigFile)
	writeTestFile(t, path, "{}")
	svc := NewBackupService(dir)
	assertReplacedByRename(t, path, "{}", func() {
		if _, err := svc.SaveConfig(model.BackupConfig{Enabled: false, Schedule: "daily"}); err != nil {
			t.Fatalf("SaveConfig: %v", err)
		}
	})
}

func TestBackupEventLogReplacesFileAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, backupEventsFile)
	writeTestFile(t, path, "[]")
	store := newBackupEventStore(dir)
	assertReplacedByRename(t, path, "[]", func() {
		if err := store.Append(model.BackupEvent{Type: "test", Message: "m"}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	})
}

func TestSaveRawSettingsReplacesFileAtomically(t *testing.T) {
	dir := t.TempDir()
	svc := NewConfigServiceWithHost(dir, NewIsolatedHostService(dir))
	first := "module.exports = { uiPort: 1880, httpAdminRoot: '/' }\n"
	doc, err := svc.SaveRawSettings(first)
	if err != nil {
		t.Fatalf("first SaveRawSettings: %v", err)
	}
	assertReplacedByRename(t, doc.Path, first, func() {
		if _, err := svc.SaveRawSettings("module.exports = { uiPort: 1881, httpAdminRoot: '/' }\n"); err != nil {
			t.Fatalf("second SaveRawSettings: %v", err)
		}
	})
}

// A DATA_DIR reached through a symlink (e.g. /home -> /var/home) must keep
// working for the files NRCC owns outside settings.js.
func TestOwnedFileWritersAcceptSymlinkedDataDir(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.MkdirAll(real, 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteDotenv(link, "A=1\n"); err != nil {
		t.Errorf("WriteDotenv: %v", err)
	}
	if _, err := NewBackupService(link).SaveConfig(model.BackupConfig{Schedule: "daily"}); err != nil {
		t.Errorf("SaveConfig: %v", err)
	}
	if err := newBackupEventStore(link).Append(model.BackupEvent{Type: "t", Message: "m"}); err != nil {
		t.Errorf("Append: %v", err)
	}
}

func TestCorruptBackupEventLogIsReplacedByEmptyLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, backupEventsFile)
	writeTestFile(t, path, `[{"id":"torn","type":"backup`)
	svc := NewBackupService(dir)

	observability, err := svc.Observability()
	if err != nil {
		t.Fatalf("Observability with a torn event log: %v", err)
	}
	if len(observability.RecentEvents) != 0 {
		t.Fatalf("RecentEvents = %+v, want empty", observability.RecentEvents)
	}

	if _, err := svc.SaveConfig(model.BackupConfig{Enabled: false, Schedule: "daily"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read event log: %v", err)
	}
	var events []model.BackupEvent
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatalf("event log still corrupt after a new event: %v (%q)", err, data)
	}
	if len(events) != 1 || events[0].Type != model.BackupEventTypeSchedulerConfig {
		t.Fatalf("events = %+v, want only the new scheduler-config event", events)
	}
}
