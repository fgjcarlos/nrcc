package security

import (
	"path/filepath"
	"testing"
	"time"

	"nrcc/internal/model"
)

func TestSessionManagerIssueVerifyAndCSRF(t *testing.T) {
	t.Parallel()

	manager, err := NewSessionManager(filepath.Join(t.TempDir(), ".session-secret"))
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	token, err := manager.Issue(model.SessionClaims{
		SID:      "ses_test",
		Sub:      "usr_test",
		Username: "alice",
		Role:     model.RoleAdmin,
		Exp:      time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	claims, err := manager.Verify(token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if claims.SID != "ses_test" || claims.Sub != "usr_test" || claims.Username != "alice" {
		t.Fatalf("Verify() claims = %+v", claims)
	}

	csrf := manager.CSRFToken(token)
	if !manager.VerifyCSRF(token, csrf) {
		t.Fatal("VerifyCSRF() = false, want true")
	}
	if manager.VerifyCSRF(token, "invalid") {
		t.Fatal("VerifyCSRF() = true for invalid token")
	}
}

func TestSessionManagerRejectsTamperedAndExpiredTokens(t *testing.T) {
	t.Parallel()

	manager, err := NewSessionManager(filepath.Join(t.TempDir(), ".session-secret"))
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	token, err := manager.Issue(model.SessionClaims{
		SID:      "ses_test",
		Sub:      "usr_test",
		Username: "alice",
		Role:     model.RoleAdmin,
		Exp:      time.Now().Add(-time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if _, err := manager.Verify(token); err == nil {
		t.Fatal("Verify() error = nil, want expired token error")
	}

	validToken, err := manager.Issue(model.SessionClaims{
		SID:      "ses_valid",
		Sub:      "usr_test",
		Username: "alice",
		Role:     model.RoleAdmin,
		Exp:      time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("Issue(valid) error = %v", err)
	}

	tampered := validToken + "x"
	if _, err := manager.Verify(tampered); err == nil {
		t.Fatal("Verify() error = nil, want tampered token error")
	}
}
