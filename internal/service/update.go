package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

type runtimeController interface {
	Start() error
	Stop() error
	Status() model.RuntimeStatus
}

type backupCoordinator interface {
	Create(reason string) (model.BackupSummary, error)
	Restore(id string, runtimeManager runtimeController) (model.BackupSummary, error)
}

type UpdateService struct {
	dataDir     string
	runner      commandRunner
	backups     backupCoordinator
	logService  *LogService
	jobsService *JobsService
}

func NewUpdateService(dataDir string, backups *BackupService) UpdateService {
	runner := platform.NewRunner()
	runner.Timeout = 2 * time.Minute
	return UpdateService{
		dataDir: dataDir,
		runner:  runner,
		backups: backups,
	}
}

// SetLogService injects the LogService for structured logging (nil-safe)
func (s *UpdateService) SetLogService(ls *LogService) {
	s.logService = ls
}

// SetJobsService injects the JobsService for job tracking (nil-safe)
func (s *UpdateService) SetJobsService(js *JobsService) {
	s.jobsService = js
}

func (s *UpdateService) Status() (model.UpdateStatus, error) {
	installed, err := s.installedVersion()
	if err != nil {
		return model.UpdateStatus{}, err
	}
	available, err := s.availableVersion()
	if err != nil {
		return model.UpdateStatus{}, err
	}
	return model.UpdateStatus{
		InstalledVersion: installed,
		AvailableVersion: available,
		UpdateAvailable:  installed != "" && available != "" && installed != available,
	}, nil
}

func (s *UpdateService) Apply(runtime runtimeController) (model.UpdateApplyResult, error) {
	status, err := s.Status()
	if err != nil {
		return model.UpdateApplyResult{}, err
	}
	if !status.UpdateAvailable {
		return model.UpdateApplyResult{
			FromVersion: status.InstalledVersion,
			ToVersion:   status.AvailableVersion,
			Message:     "Node-RED is already up to date.",
		}, nil
	}

	// Start job tracking if available
	var jobCtx *JobContext
	if s.jobsService != nil {
		var jobErr error
		jobCtx, jobErr = NewJobContext(s.jobsService, s.logService, model.JobTypeUpdateApply, "system", fmt.Sprintf("Updating Node-RED from %s to %s", status.InstalledVersion, status.AvailableVersion))
		if jobErr != nil {
			return model.UpdateApplyResult{}, fmt.Errorf("start update job: %w", jobErr)
		}
		defer func() {
			if jobCtx != nil && err != nil {
				_ = jobCtx.Fail(err.Error())
			}
		}()
	}

	preventive, err := s.backups.Create("pre_update")
	if err != nil {
		return model.UpdateApplyResult{}, fmt.Errorf("create preventive backup: %w", err)
	}

	wasRunning := runtime != nil && runtime.Status().Running
	if runtime != nil && wasRunning {
		if err := runtime.Stop(); err != nil {
			return model.UpdateApplyResult{}, fmt.Errorf("stop runtime before update: %w", err)
		}
	}

	if _, err := s.runner.Run(s.dataDir, "npm", "install", "node-red@"+status.AvailableVersion); err != nil {
		rollbackErr := s.rollback(preventive.ID, runtime)
		if rollbackErr != nil {
			return model.UpdateApplyResult{}, fmt.Errorf("update failed: %v; rollback failed: %w", err, rollbackErr)
		}
		return model.UpdateApplyResult{
			FromVersion:        status.InstalledVersion,
			ToVersion:          status.AvailableVersion,
			PreventiveBackupID: preventive.ID,
			RolledBack:         true,
			Message:            "Update failed and the previous state was restored.",
		}, fmt.Errorf("apply update: %w", err)
	}

	if runtime != nil {
		if err := runtime.Start(); err != nil {
			rollbackErr := s.rollback(preventive.ID, runtime)
			if rollbackErr != nil {
				return model.UpdateApplyResult{}, fmt.Errorf("restart after update failed: %v; rollback failed: %w", err, rollbackErr)
			}
			return model.UpdateApplyResult{
				FromVersion:        status.InstalledVersion,
				ToVersion:          status.AvailableVersion,
				PreventiveBackupID: preventive.ID,
				RolledBack:         true,
				Message:            "Update failed during restart and the previous state was restored.",
			}, fmt.Errorf("restart after update: %w", err)
		}
		if err := waitForHealthy(runtime, 10*time.Second); err != nil {
			rollbackErr := s.rollback(preventive.ID, runtime)
			if rollbackErr != nil {
				return model.UpdateApplyResult{}, fmt.Errorf("health verification failed: %v; rollback failed: %w", err, rollbackErr)
			}
			return model.UpdateApplyResult{
				FromVersion:        status.InstalledVersion,
				ToVersion:          status.AvailableVersion,
				PreventiveBackupID: preventive.ID,
				RolledBack:         true,
				Message:            "Updated runtime failed health checks and was rolled back.",
			}, fmt.Errorf("verify updated runtime health: %w", err)
		}
		if !wasRunning {
			if err := runtime.Stop(); err != nil {
				return model.UpdateApplyResult{}, fmt.Errorf("stop verification runtime: %w", err)
			}
		}
	}

	// Emit log event
	if s.logService != nil {
		entry := model.LogEntry{
			Level:     model.LogLevelInfo,
			Source:    model.SourceUpdate,
			Event:     model.EventUpdateApply,
			Message:   fmt.Sprintf("Node-RED updated from %s to %s", status.InstalledVersion, status.AvailableVersion),
			Timestamp: time.Now().UTC(),
			Metadata: map[string]any{
				"fromVersion": status.InstalledVersion,
				"toVersion":   status.AvailableVersion,
				"backupId":    preventive.ID,
			},
		}
		_ = s.logService.Write(entry)
	}

	// Complete job
	if jobCtx != nil {
		_ = jobCtx.Complete(fmt.Sprintf("Node-RED updated to %s", status.AvailableVersion))
	}

	return model.UpdateApplyResult{
		FromVersion:        status.InstalledVersion,
		ToVersion:          status.AvailableVersion,
		PreventiveBackupID: preventive.ID,
		RolledBack:         false,
		Message:            "Node-RED updated successfully.",
	}, nil
}

func (s *UpdateService) installedVersion() (string, error) {
	output, err := s.runner.Run(s.dataDir, "npm", "ls", "node-red", "--depth=0", "--json")
	if err != nil {
		return "", fmt.Errorf("read installed node-red version: %w", err)
	}

	var payload struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return "", fmt.Errorf("parse installed node-red version: %w", err)
	}

	if dep, ok := payload.Dependencies["node-red"]; ok {
		return strings.TrimSpace(dep.Version), nil
	}
	return "", nil
}

func (s *UpdateService) availableVersion() (string, error) {
	output, err := s.runner.Run(s.dataDir, "npm", "view", "node-red", "version")
	if err != nil {
		return "", fmt.Errorf("read available node-red version: %w", err)
	}
	return strings.TrimSpace(output), nil
}

func (s *UpdateService) rollback(backupID string, runtime runtimeController) error {
	if _, err := s.backups.Restore(backupID, runtime); err != nil {
		return fmt.Errorf("restore preventive backup: %w", err)
	}
	if runtime != nil {
		if !runtime.Status().Running {
			if err := runtime.Start(); err != nil {
				return fmt.Errorf("restart runtime after rollback: %w", err)
			}
		}
		if err := waitForHealthy(runtime, 10*time.Second); err != nil {
			return fmt.Errorf("verify rolled back runtime health: %w", err)
		}
	}
	return nil
}

func waitForHealthy(runtime runtimeController, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status := runtime.Status()
		if status.Running && status.Healthy {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("runtime did not become healthy within %s", timeout)
}
