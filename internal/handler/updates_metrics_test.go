package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
)

// stubUpdateMetrics is a test double for updateMetricsRecorder.
type stubUpdateMetrics struct {
	calls []bool
}

func (s *stubUpdateMetrics) RecordUpdateAttempt(success bool) {
	s.calls = append(s.calls, success)
}

// TestPostApply_RecordsUpdateAttempt verifies that PostApply records a metric call
// after the async update flow settles.
func TestPostApply_RecordsUpdateAttempt(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	h := NewUpdateHandler(svc)
	stub := &stubUpdateMetrics{}
	h.SetUpdateMetrics(stub)

	req := httptest.NewRequest(http.MethodPost, "/api/updates/apply", nil)
	rec := httptest.NewRecorder()

	h.PostApply(rec, req)
	waitForUpdateFlowToSettle(t, svc)

	// The metric must be recorded exactly once after the flow settles.
	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 RecordUpdateAttempt call, got %d", len(stub.calls))
	}
}

// TestPostApply_RecordsCorrectSuccessFlag verifies that the success flag matches
// the actual flow outcome (triangulation).
func TestPostApply_RecordsCorrectSuccessFlag(t *testing.T) {
	tmpDir := t.TempDir()
	installFakeNPM(t)
	svc := service.NewUpdateService(tmpDir)
	h := NewUpdateHandler(svc)
	stub := &stubUpdateMetrics{}
	h.SetUpdateMetrics(stub)

	req := httptest.NewRequest(http.MethodPost, "/api/updates/apply", nil)
	rec := httptest.NewRecorder()

	h.PostApply(rec, req)
	waitForUpdateFlowToSettle(t, svc)

	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 RecordUpdateAttempt call, got %d", len(stub.calls))
	}

	// The recorded success flag must match the actual final flow state.
	finalState := svc.GetFlowState()
	expectedSuccess := finalState.State == model.StateCompleted
	if stub.calls[0] != expectedSuccess {
		t.Fatalf("RecordUpdateAttempt(%v) but flow state is %s", stub.calls[0], finalState.State)
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
