package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

type securityPostureSnapshotStub struct {
	posture service.SecurityPosture
	err     error
}

func (s securityPostureSnapshotStub) Snapshot(time.Time) (service.SecurityPosture, error) {
	return s.posture, s.err
}

func TestSecurityPostureHandlerGet(t *testing.T) {
	t.Parallel()

	want := service.SecurityPosture{
		EncryptionKeyConfigured: true,
		BackupDownloadAdminOnly: true,
		ActiveRefreshSessions:   3,
		MFA: service.SecurityPostureMFA{EnrolledAdmins: 1, TotalAdmins: 2},
	}
	h := NewSecurityPostureHandler(securityPostureSnapshotStub{posture: want})
	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest(http.MethodGet, "/api/system/security-posture", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var response model.ApiResponse[service.SecurityPosture]
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data != want {
		t.Fatalf("response = %+v, want posture %+v", response, want)
	}
}

func TestSecurityPostureHandlerGetServiceError(t *testing.T) {
	t.Parallel()

	h := NewSecurityPostureHandler(securityPostureSnapshotStub{err: errors.New("store unavailable")})
	rec := httptest.NewRecorder()
	h.Get(rec, httptest.NewRequest(http.MethodGet, "/api/system/security-posture", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var response model.ApiErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Success || response.Error == nil || response.Error.Code != "SECURITY_POSTURE_ERROR" {
		t.Fatalf("response = %+v, want SECURITY_POSTURE_ERROR", response)
	}
}
