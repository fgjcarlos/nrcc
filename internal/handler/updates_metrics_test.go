package handler

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
)

type stubUpdateMetrics struct {
	mu   sync.Mutex
	calls []bool
	done  chan struct{}
}

func newStubUpdateMetrics() *stubUpdateMetrics {
	return &stubUpdateMetrics{done: make(chan struct{}, 1)}
}

func (s *stubUpdateMetrics) RecordUpdateAttempt(success bool) {
	s.mu.Lock()
	s.calls = append(s.calls, success)
	s.mu.Unlock()
	select {
	case s.done <- struct{}{}:
	default:
	}
}

func (s *stubUpdateMetrics) waitForCall(t *testing.T) {
	t.Helper()
	select {
	case <-s.done:
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for RecordUpdateAttempt call")
	}
}

func (s *stubUpdateMetrics) getCalls() []bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]bool, len(s.calls))
	copy(result, s.calls)
	return result
}

// TestPostApply_RecordsUpdateAttempt verifies that PostApply records a metric call
// after the async update flow settles.
func TestPostApply_RecordsUpdateAttempt(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	h := NewUpdateHandler(svc)
	stub := newStubUpdateMetrics()
	h.SetUpdateMetrics(stub)

	req := httptest.NewRequest(http.MethodPost, "/api/updates/apply", nil)
	rec := httptest.NewRecorder()

	h.PostApply(rec, req)
	stub.waitForCall(t)

	calls := stub.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 RecordUpdateAttempt call, got %d", len(calls))
	}
}

// TestPostApply_RecordsCorrectSuccessFlag verifies that the success flag matches
// the actual flow outcome (triangulation).
func TestPostApply_RecordsCorrectSuccessFlag(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	h := NewUpdateHandler(svc)
	stub := newStubUpdateMetrics()
	h.SetUpdateMetrics(stub)

	req := httptest.NewRequest(http.MethodPost, "/api/updates/apply", nil)
	rec := httptest.NewRecorder()

	h.PostApply(rec, req)
	stub.waitForCall(t)

	calls := stub.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 RecordUpdateAttempt call, got %d", len(calls))
	}

	finalState := svc.GetFlowState()
	expectedSuccess := finalState.State == model.StateCompleted
	if calls[0] != expectedSuccess {
		t.Fatalf("RecordUpdateAttempt(%v) but flow state is %s", calls[0], finalState.State)
	}
}

// TestUpdate_NoMetricsNilGuard verifies that PostApply works when no metrics recorder
// is set (nil guard must not panic).
func TestUpdate_NoMetricsNilGuard(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	h := NewUpdateHandler(svc)
	// updateMetrics is nil (no SetUpdateMetrics call)

	req := httptest.NewRequest(http.MethodPost, "/api/updates/apply", nil)
	rec := httptest.NewRecorder()

	// Must not panic.
	h.PostApply(rec, req)
	waitForUpdateFlowToSettle(t, svc)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
