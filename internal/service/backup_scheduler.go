package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/robfig/cron/v3"
)

var backupCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

type backupScheduler struct {
	svc *BackupService

	mu      sync.RWMutex
	engine  *cron.Cron
	entryID cron.EntryID
	started bool
	status  model.BackupSchedulerStatus
}

func newBackupScheduler(svc *BackupService) *backupScheduler {
	return &backupScheduler{
		svc: svc,
		engine: cron.New(
			cron.WithParser(backupCronParser),
			cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)),
		),
		status: model.BackupSchedulerStatus{
			Schedule: defaultBackupConfig.Schedule,
		},
	}
}

func (s *backupScheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.mu.Unlock()

	s.engine.Start()

	go func() {
		<-ctx.Done()
		stopCtx := s.engine.Stop()
		select {
		case <-stopCtx.Done():
		case <-time.After(5 * time.Second):
		}
	}()

	cfg, err := s.svc.GetConfig()
	if err != nil {
		s.updateError(fmt.Sprintf("failed to load backup config: %v", err))
		s.svc.recordEvent(model.BackupEvent{
			Type:    model.BackupEventTypeSchedulerError,
			Status:  "error",
			Message: "Backup scheduler failed to load config",
			Trigger: "startup",
			Error:   err.Error(),
		})
		return
	}

	if err := s.ApplyConfig(cfg); err != nil {
		s.updateError(err.Error())
	}
}

func (s *backupScheduler) ApplyConfig(cfg model.BackupConfig) error {
	normalized := normalizeBackupConfig(cfg)
	spec, err := scheduleSpec(normalized)

	s.mu.Lock()
	defer s.mu.Unlock()

	previous := s.status
	if s.entryID != 0 {
		s.engine.Remove(s.entryID)
		s.entryID = 0
	}

	s.status = model.BackupSchedulerStatus{
		Enabled:        normalized.Enabled,
		Scheduled:      false,
		Schedule:       normalized.Schedule,
		CustomSchedule: normalized.CustomSchedule,
		LastRunAt:      previous.LastRunAt,
		LastSuccessAt:  previous.LastSuccessAt,
		LastBackupID:   previous.LastBackupID,
		LastError:      "",
	}

	if err != nil {
		s.status.LastError = err.Error()
		s.svc.recordEvent(model.BackupEvent{
			Type:     model.BackupEventTypeSchedulerError,
			Status:   "error",
			Message:  "Backup scheduler configuration is invalid",
			Schedule: normalized.Schedule,
			Trigger:  "config-apply",
			Error:    err.Error(),
		})
		return err
	}

	if spec == "" {
		s.status.ActiveSpec = ""
		s.status.NextRunAt = ""
		return nil
	}

	entryID, addErr := s.engine.AddFunc(spec, s.runAutomaticBackup)
	if addErr != nil {
		wrapped := fmt.Errorf("failed to schedule automatic backup: %w", addErr)
		s.status.LastError = wrapped.Error()
		s.svc.recordEvent(model.BackupEvent{
			Type:       model.BackupEventTypeSchedulerError,
			Status:     "error",
			Message:    "Automatic backup scheduling failed",
			Schedule:   normalized.Schedule,
			ActiveSpec: spec,
			Trigger:    "config-apply",
			Error:      wrapped.Error(),
		})
		return wrapped
	}

	s.entryID = entryID
	s.status.Scheduled = true
	s.status.ActiveSpec = spec
	s.status.NextRunAt = s.nextRunAtLocked()
	s.svc.recordEvent(model.BackupEvent{
		Type:       model.BackupEventTypeSchedulerConfig,
		Status:     "success",
		Message:    "Backup scheduler armed",
		Schedule:   normalized.Schedule,
		ActiveSpec: spec,
		Trigger:    "config-apply",
	})

	return nil
}

func (s *backupScheduler) Status() model.BackupSchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *backupScheduler) runAutomaticBackup() {
	now := time.Now().UTC().Format(time.RFC3339)

	s.mu.Lock()
	s.status.LastRunAt = now
	s.status.LastError = ""
	s.mu.Unlock()

	backup, err := s.svc.CreateTyped(model.BackupTypeAuto, "")
	if err != nil {
		s.mu.Lock()
		s.status.LastError = fmt.Sprintf("automatic backup failed: %v", err)
		s.status.NextRunAt = s.nextRunAtLocked()
		s.mu.Unlock()
		s.svc.recordEvent(model.BackupEvent{
			Type:       model.BackupEventTypeSchedulerRun,
			Status:     "error",
			OccurredAt: now,
			Message:    "Automatic backup run failed",
			Schedule:   s.Status().Schedule,
			ActiveSpec: s.Status().ActiveSpec,
			Trigger:    "scheduler",
			Error:      err.Error(),
		})
		return
	}

	s.mu.Lock()
	s.status.LastSuccessAt = now
	s.status.LastBackupID = backup.ID
	s.status.NextRunAt = s.nextRunAtLocked()
	s.mu.Unlock()
	status := s.Status()
	s.svc.recordEvent(model.BackupEvent{
		Type:       model.BackupEventTypeSchedulerRun,
		Status:     "success",
		OccurredAt: now,
		BackupID:   backup.ID,
		BackupName: backup.Name,
		BackupType: backup.Type,
		Message:    "Automatic backup run completed",
		Schedule:   status.Schedule,
		ActiveSpec: status.ActiveSpec,
		Trigger:    "scheduler",
	})
}

func (s *backupScheduler) updateError(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.LastError = message
}

func (s *backupScheduler) nextRunAtLocked() string {
	if s.entryID == 0 {
		return ""
	}

	next := s.engine.Entry(s.entryID).Next
	if next.IsZero() {
		return ""
	}

	return next.UTC().Format(time.RFC3339)
}
