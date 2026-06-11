package service

import (
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/store"
)

func newTestAuthService(t *testing.T) *AuthService {
	t.Helper()
	dir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](dir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](dir + "/sessions.json")
	return NewAuthService("test-secret", userStore, sessionStore)
}

func TestGenerateToken_ShortLived(t *testing.T) {
	svc := newTestAuthService(t)

	user := &model.CCUser{ID: "u1", Username: "admin", Role: model.RoleAdmin}
	tokenStr, err := svc.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims, err := svc.VerifyToken(tokenStr)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}

	ttl := time.Unix(claims.ExpiresAt, 0).Sub(time.Unix(claims.IssuedAt, 0))
	if ttl > 16*time.Minute || ttl < 14*time.Minute {
		t.Fatalf("expected ~15min TTL, got %v", ttl)
	}
}

func TestRefreshSession_CreateAndValidate(t *testing.T) {
	svc := newTestAuthService(t)

	token, err := svc.CreateRefreshSession("u1")
	if err != nil {
		t.Fatalf("CreateRefreshSession: %v", err)
	}
	if len(token) < 32 {
		t.Fatalf("token too short: %d", len(token))
	}

	sess, err := svc.ValidateRefreshSession(token)
	if err != nil {
		t.Fatalf("ValidateRefreshSession: %v", err)
	}
	if sess.UserID != "u1" {
		t.Fatalf("expected userID u1, got %s", sess.UserID)
	}
}

func TestRefreshSession_RevokeBlocksValidation(t *testing.T) {
	svc := newTestAuthService(t)

	token, _ := svc.CreateRefreshSession("u1")

	if err := svc.RevokeRefreshSession(token); err != nil {
		t.Fatalf("RevokeRefreshSession: %v", err)
	}

	_, err := svc.ValidateRefreshSession(token)
	if err == nil {
		t.Fatal("expected error for revoked session")
	}
}

func TestRefreshSession_RevokeUserSessions(t *testing.T) {
	svc := newTestAuthService(t)

	tok1, _ := svc.CreateRefreshSession("u1")
	tok2, _ := svc.CreateRefreshSession("u1")
	tok3, _ := svc.CreateRefreshSession("u2")

	if err := svc.RevokeUserSessions("u1"); err != nil {
		t.Fatalf("RevokeUserSessions: %v", err)
	}

	if _, err := svc.ValidateRefreshSession(tok1); err == nil {
		t.Fatal("expected tok1 revoked")
	}
	if _, err := svc.ValidateRefreshSession(tok2); err == nil {
		t.Fatal("expected tok2 revoked")
	}
	if _, err := svc.ValidateRefreshSession(tok3); err != nil {
		t.Fatalf("tok3 should still be valid: %v", err)
	}
}

func TestRefreshSession_InvalidTokenNotFound(t *testing.T) {
	svc := newTestAuthService(t)

	_, err := svc.ValidateRefreshSession("nonexistent-token")
	if err == nil {
		t.Fatal("expected error for unknown token")
	}
}
