package service

import (
	"errors"
	"testing"

	"nrcc/internal/model"
)

type fakeBackups struct {
	created  []string
	restored []string
}

func (f *fakeBackups) Create(reason string) (model.BackupSummary, error) {
	f.created = append(f.created, reason)
	return model.BackupSummary{ID: "bkp_test", Reason: reason}, nil
}

func (f *fakeBackups) Restore(id string, _ runtimeController) (model.BackupSummary, error) {
	f.restored = append(f.restored, id)
	return model.BackupSummary{ID: id, Reason: "pre_restore"}, nil
}

type fakeRuntime struct {
	running bool
	healthy bool
	starts  int
	stops   int
}

func (f *fakeRuntime) Start() error {
	f.running = true
	f.healthy = true
	f.starts++
	return nil
}

func (f *fakeRuntime) Stop() error {
	f.running = false
	f.stops++
	return nil
}

func (f *fakeRuntime) Status() model.RuntimeStatus {
	return model.RuntimeStatus{
		Running: f.running,
		Healthy: f.healthy,
		Version: "4.0.0",
	}
}

func TestUpdateServiceStatus(t *testing.T) {
	t.Parallel()

	backups := &fakeBackups{}
	service := UpdateService{
		dataDir: t.TempDir(),
		runner: fakeRunner{
			outputs: map[string]string{
				"npm ls node-red --depth=0 --json": `{"dependencies":{"node-red":{"version":"4.0.0"}}}`,
				"npm view node-red version":        "4.1.0",
			},
		},
		backups: backups,
	}

	status, err := service.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.InstalledVersion != "4.0.0" || status.AvailableVersion != "4.1.0" || !status.UpdateAvailable {
		t.Fatalf("Status() = %+v", status)
	}
}

func TestUpdateServiceApplySuccess(t *testing.T) {
	t.Parallel()

	backups := &fakeBackups{}
	runtime := &fakeRuntime{running: true, healthy: true}
	service := UpdateService{
		dataDir: t.TempDir(),
		runner: fakeRunner{
			outputs: map[string]string{
				"npm ls node-red --depth=0 --json": `{"dependencies":{"node-red":{"version":"4.0.0"}}}`,
				"npm view node-red version":        "4.1.0",
				"npm install node-red@4.1.0":       "updated",
			},
		},
		backups: backups,
	}

	result, err := service.Apply(runtime)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.PreventiveBackupID != "bkp_test" || result.RolledBack {
		t.Fatalf("Apply() result = %+v", result)
	}
	if len(backups.created) != 1 || backups.created[0] != "pre_update" {
		t.Fatalf("Create() calls = %v", backups.created)
	}
	if runtime.starts != 1 || runtime.stops != 1 {
		t.Fatalf("runtime start/stop = %d/%d", runtime.starts, runtime.stops)
	}
}

func TestUpdateServiceApplyRollbackOnInstallFailure(t *testing.T) {
	t.Parallel()

	backups := &fakeBackups{}
	runtime := &fakeRuntime{running: true, healthy: true}
	service := UpdateService{
		dataDir: t.TempDir(),
		runner: fakeRunner{
			outputs: map[string]string{
				"npm ls node-red --depth=0 --json": `{"dependencies":{"node-red":{"version":"4.0.0"}}}`,
				"npm view node-red version":        "4.1.0",
			},
			errors: map[string]error{
				"npm install node-red@4.1.0": errors.New("boom"),
			},
		},
		backups: backups,
	}

	if _, err := service.Apply(runtime); err == nil {
		t.Fatal("Apply() error = nil")
	}
	if len(backups.restored) != 1 || backups.restored[0] != "bkp_test" {
		t.Fatalf("Restore() calls = %v", backups.restored)
	}
}
