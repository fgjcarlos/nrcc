package service

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/store"
)

func TestAuthServiceCountActiveRefreshSessions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sessions := store.NewJSONStore[model.RefreshSessions](dir + "/sessions.json")
	svc := NewAuthService("secret", store.NewJSONStore[model.CCUsers](dir+"/users.json"), sessions)
	now := time.Unix(1_700_000_000, 0)
	if err := sessions.Write(model.RefreshSessions{Sessions: []model.RefreshSession{
		{ID: "active", ExpiresAt: now.Add(time.Hour).Unix()},
		{ID: "expired", ExpiresAt: now.Add(-time.Second).Unix()},
		{ID: "revoked", ExpiresAt: now.Add(time.Hour).Unix(), Revoked: true},
	}}); err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	got, err := svc.CountActiveRefreshSessions(now)
	if err != nil {
		t.Fatalf("CountActiveRefreshSessions: %v", err)
	}
	if got != 1 {
		t.Fatalf("active sessions = %d, want 1", got)
	}
}

func TestAuthServiceCountActiveRefreshSessionsEmptyAndStoreError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	svc := NewAuthService("secret", store.NewJSONStore[model.CCUsers](dir+"/users.json"), store.NewJSONStore[model.RefreshSessions](dir+"/sessions.json"))
	got, err := svc.CountActiveRefreshSessions(time.Now())
	if err != nil || got != 0 {
		t.Fatalf("missing store result = (%d, %v), want (0, nil)", got, err)
	}

	brokenPath := dir + "/broken.json"
	if err := os.WriteFile(brokenPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("write malformed store: %v", err)
	}
	broken := NewAuthService("secret", store.NewJSONStore[model.CCUsers](dir+"/users.json"), store.NewJSONStore[model.RefreshSessions](brokenPath))
	if _, err := broken.CountActiveRefreshSessions(time.Now()); err == nil {
		t.Fatal("expected malformed session store error")
	}
}

func TestMfaServiceCountEnrolled(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	auth := NewAuthService("secret", store.NewJSONStore[model.CCUsers](dir+"/users.json"), store.NewJSONStore[model.RefreshSessions](dir+"/sessions.json"))
	svc := NewMfaService(dir, auth)
	if err := svc.store.Write(model.MfaStore{Enrollments: []model.MfaEnrollment{
		{UserID: "enrolled"},
		{UserID: "pending", Pending: true},
		{UserID: "viewer"},
	}}); err != nil {
		t.Fatalf("seed MFA store: %v", err)
	}

	got, err := svc.CountEnrolled([]string{"enrolled", "pending", "missing"})
	if err != nil {
		t.Fatalf("CountEnrolled: %v", err)
	}
	if got != 1 {
		t.Fatalf("enrolled admins = %d, want 1", got)
	}
}

func TestMfaServiceCountEnrolledEmptyAndStoreError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	auth := NewAuthService("secret", store.NewJSONStore[model.CCUsers](dir+"/users.json"), store.NewJSONStore[model.RefreshSessions](dir+"/sessions.json"))
	svc := NewMfaService(dir, auth)
	got, err := svc.CountEnrolled(nil)
	if err != nil || got != 0 {
		t.Fatalf("missing store result = (%d, %v), want (0, nil)", got, err)
	}
	if err := os.WriteFile(dir+"/mfa.json", []byte("{"), 0o600); err != nil {
		t.Fatalf("write malformed MFA store: %v", err)
	}
	if _, err := svc.CountEnrolled([]string{"admin"}); err == nil {
		t.Fatal("expected malformed MFA store error")
	}
}

type postureAuthStub struct {
	sessions int
	users    []model.CCUser
	err      error
}

func (s postureAuthStub) CountActiveRefreshSessions(time.Time) (int, error) {
	return s.sessions, s.err
}

func (s postureAuthStub) GetAllUsers() ([]model.CCUser, error) {
	return s.users, s.err
}

type postureMfaStub struct {
	enrolled int
	err      error
}

func (s postureMfaStub) CountEnrolled([]string) (int, error) { return s.enrolled, s.err }

func TestSecurityPostureServiceSnapshot(t *testing.T) {
	t.Parallel()

	svc := NewSecurityPostureService(true, true,
		postureAuthStub{sessions: 2, users: []model.CCUser{
			{ID: "a1", Role: model.RoleAdmin},
			{ID: "a2", Role: model.RoleAdmin},
			{ID: "v1", Role: model.RoleViewer},
		}},
		postureMfaStub{enrolled: 1},
	)

	got, err := svc.Snapshot(time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	want := SecurityPosture{
		EncryptionKeyConfigured: true,
		BackupDownloadAdminOnly: true,
		ActiveRefreshSessions:   2,
		MFA: SecurityPostureMFA{EnrolledAdmins: 1, TotalAdmins: 2},
	}
	if got != want {
		t.Fatalf("Snapshot = %+v, want %+v", got, want)
	}
}

func TestSecurityPostureServiceSnapshotPropagatesReaderErrors(t *testing.T) {
	t.Parallel()

	readErr := errors.New("read failed")
	tests := []struct {
		name string
		auth postureAuthStub
		mfa  postureMfaStub
	}{
		{name: "auth", auth: postureAuthStub{err: readErr}},
		{name: "mfa", auth: postureAuthStub{}, mfa: postureMfaStub{err: readErr}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSecurityPostureService(false, true, tt.auth, tt.mfa)
			if _, err := svc.Snapshot(time.Now()); !errors.Is(err, readErr) {
				t.Fatalf("Snapshot error = %v, want %v", err, readErr)
			}
		})
	}
}
