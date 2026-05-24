package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubLoginMetrics is a test double for loginMetricsRecorder.
type stubLoginMetrics struct {
	calls []bool
}

func (s *stubLoginMetrics) RecordLoginAttempt(success bool) {
	s.calls = append(s.calls, success)
}

// TestLogin_RecordsFailureMetric verifies that a failed login (wrong password)
// calls RecordLoginAttempt(false) exactly once.
func TestLogin_RecordsFailureMetric(t *testing.T) {
	h, _ := setupAuthTest(t)
	stub := &stubLoginMetrics{}
	h.SetLoginMetrics(stub)

	body := `{"username":"admin","password":"wrong-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 metric call, got %d", len(stub.calls))
	}
	if stub.calls[0] != false {
		t.Fatalf("expected RecordLoginAttempt(false) for failed login")
	}
}

// TestLogin_RecordsSuccessMetric verifies that a successful login
// calls RecordLoginAttempt(true) exactly once.
func TestLogin_RecordsSuccessMetric(t *testing.T) {
	h, _ := setupAuthTest(t)
	stub := &stubLoginMetrics{}
	h.SetLoginMetrics(stub)

	body := `{"username":"admin","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(stub.calls) != 1 {
		t.Fatalf("expected 1 metric call, got %d", len(stub.calls))
	}
	if stub.calls[0] != true {
		t.Fatalf("expected RecordLoginAttempt(true) for successful login")
	}
}

// TestLogin_NoMetricsNilGuard verifies that Login works correctly when no
// metrics recorder is set (nil guard must not panic).
func TestLogin_NoMetricsNilGuard(t *testing.T) {
	h, _ := setupAuthTest(t)
	// Do NOT call SetLoginMetrics — loginMetrics stays nil.

	body := `{"username":"admin","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Must not panic.
	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
