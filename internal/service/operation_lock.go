package service

import (
	"fmt"
	"sync"
	"time"

	"nrcc/internal/model"
)

type OperationLock struct {
	mu         sync.Mutex
	active     bool
	opType     string
	detail     string
	started    time.Time
	logService *LogService
}

func NewOperationLock() *OperationLock {
	return &OperationLock{}
}

// SetLogService injects the LogService for structured logging (nil-safe)
func (l *OperationLock) SetLogService(ls *LogService) {
	l.logService = ls
}

func (l *OperationLock) Acquire(opType, detail string) (func(), error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.active {
		return nil, fmt.Errorf("system busy with %s", l.opType)
	}

	l.active = true
	l.opType = opType
	l.detail = detail
	l.started = time.Now().UTC()

	// Emit lock acquired event
	if l.logService != nil {
		entry := model.LogEntry{
			Level:     model.LogLevelInfo,
			Source:    model.SourceOperation,
			Event:     model.EventOperationLocked,
			Message:   fmt.Sprintf("Operation locked: %s", opType),
			Timestamp: time.Now().UTC(),
		}
		_ = l.logService.Write(entry)
	}

	return func() {
		l.mu.Lock()
		defer l.mu.Unlock()

		// Emit lock released event
		if l.logService != nil {
			entry := model.LogEntry{
				Level:     model.LogLevelInfo,
				Source:    model.SourceOperation,
				Event:     model.EventOperationReleased,
				Message:   fmt.Sprintf("Operation released: %s", l.opType),
				Timestamp: time.Now().UTC(),
			}
			_ = l.logService.Write(entry)
		}

		l.active = false
		l.opType = ""
		l.detail = ""
		l.started = time.Time{}
	}, nil
}

func (l *OperationLock) Status() model.OperationStatus {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.active {
		return model.OperationStatus{}
	}

	return model.OperationStatus{
		Busy:      true,
		Type:      l.opType,
		Detail:    l.detail,
		StartedAt: l.started.Format(time.RFC3339),
	}
}
