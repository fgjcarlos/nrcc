package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/google/uuid"
)

const (
	backupEventsFile = "backup_events.json"
	maxBackupEvents  = 40
)

type backupEventStore struct {
	dataDir string
	mu      sync.Mutex
}

func newBackupEventStore(dataDir string) *backupEventStore {
	return &backupEventStore{dataDir: dataDir}
}

func (s *backupEventStore) List() ([]model.BackupEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *backupEventStore) Append(event model.BackupEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	events, err := s.loadLocked()
	if err != nil {
		return err
	}

	event.ID = firstNonEmpty(event.ID, uuid.NewString())
	event.OccurredAt = firstNonEmpty(event.OccurredAt, time.Now().UTC().Format(time.RFC3339))
	if event.Status == "" {
		event.Status = "info"
	}

	events = append([]model.BackupEvent{event}, events...)
	if len(events) > maxBackupEvents {
		events = events[:maxBackupEvents]
	}

	return s.saveLocked(events)
}

func (s *backupEventStore) loadLocked() ([]model.BackupEvent, error) {
	path := filepath.Join(s.dataDir, backupEventsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []model.BackupEvent{}, nil
		}
		return nil, fmt.Errorf("read backup event log: %w", err)
	}

	var events []model.BackupEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("parse backup event log: %w", err)
	}

	sort.SliceStable(events, func(i, j int) bool {
		return events[i].OccurredAt > events[j].OccurredAt
	})

	if len(events) > maxBackupEvents {
		events = events[:maxBackupEvents]
	}

	return events, nil
}

func (s *backupEventStore) saveLocked(events []model.BackupEvent) error {
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir for backup event log: %w", err)
	}

	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return fmt.Errorf("encode backup event log: %w", err)
	}

	path := filepath.Join(s.dataDir, backupEventsFile)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write backup event log: %w", err)
	}

	return nil
}
