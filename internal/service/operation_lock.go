package service

import (
	"fmt"
	"sync"
	"time"

	"nrcc/internal/model"
)

type OperationLock struct {
	mu      sync.Mutex
	active  bool
	opType  string
	detail  string
	started time.Time
}

func NewOperationLock() *OperationLock {
	return &OperationLock{}
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

	return func() {
		l.mu.Lock()
		defer l.mu.Unlock()
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
