package service

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/store"
)

func newAtomicAuthService(t *testing.T) (*AuthService, *store.JSONStore[model.CCUsers], *store.JSONStore[model.RefreshSessions]) {
	t.Helper()
	dir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](dir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](dir + "/sessions.json")
	return NewAuthService("test-secret", userStore, sessionStore), userStore, sessionStore
}

func TestAuthService_UpdateUser_ConcurrentFields(t *testing.T) {
	svc, userStore, _ := newAtomicAuthService(t)
	user := &model.CCUser{ID: "u1", Username: "before", PasswordHash: "old", Role: model.RoleViewer, CreatedAt: "created", UpdatedAt: "before"}
	if err := svc.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	errs := runConcurrent(t, 4, func(i int) error {
		return svc.UpdateUser(user.ID, func(current *model.CCUser) error {
			switch i {
			case 0:
				current.Username = "after"
			case 1:
				current.PasswordHash = "new-hash"
			case 2:
				current.Role = model.RoleAdmin
			case 3:
				current.UpdatedAt = "after"
			}
			return nil
		})
	})
	assertNoError(t, errs)

	users, err := userStore.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	got := users.Users[0]
	if got.Username != "after" || got.PasswordHash != "new-hash" || got.Role != model.RoleAdmin || got.UpdatedAt != "after" {
		t.Fatalf("concurrent field mutations were lost: %+v", got)
	}
}

func TestAuthService_DeleteUser_Concurrent(t *testing.T) {
	svc, userStore, _ := newAtomicAuthService(t)
	for _, user := range []*model.CCUser{
		{ID: "admin", Username: "admin", Role: model.RoleAdmin},
		{ID: "target", Username: "target", Role: model.RoleViewer},
	} {
		if err := svc.CreateUser(user); err != nil {
			t.Fatalf("CreateUser(%s): %v", user.ID, err)
		}
	}

	errs := runConcurrent(t, 20, func(int) error { return svc.DeleteUser("target") })
	succeeded, notFound := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrUserNotFound):
			notFound++
		default:
			t.Fatalf("unexpected delete error: %v", err)
		}
	}
	if succeeded != 1 || notFound != len(errs)-1 {
		t.Fatalf("delete outcomes: succeeded=%d notFound=%d", succeeded, notFound)
	}
	users, err := userStore.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(users.Users) != 1 || users.Users[0].ID != "admin" {
		t.Fatalf("unexpected users after delete: %+v", users.Users)
	}
}

func TestAuthService_CreateRefreshSession_Concurrent(t *testing.T) {
	svc, _, sessionStore := newAtomicAuthService(t)
	const n = 50
	tokens := make([]string, n)
	errs := runConcurrent(t, n, func(i int) error {
		var err error
		tokens[i], err = svc.CreateRefreshSession(fmt.Sprintf("user-%d", i))
		return err
	})
	assertNoError(t, errs)

	sessions, err := sessionStore.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(sessions.Sessions) != n {
		t.Fatalf("persisted sessions=%d, want %d", len(sessions.Sessions), n)
	}
	// #669: the persisted id is sha256(token), not the token itself. The
	// raw token is returned to the client and never written to disk, so
	// checking it is absent on disk and its hash is present verifies the
	// new contract.
	seen := make(map[string]bool, n)
	for _, session := range sessions.Sessions {
		seen[session.ID] = true
	}
	for _, token := range tokens {
		if token == "" {
			t.Fatalf("empty token returned")
		}
		if seen[token] {
			t.Fatalf("raw token %q was persisted as session id (#669)", token)
		}
		tokenHash := hashRefreshToken(token)
		if !seen[tokenHash] {
			t.Fatalf("hash of token %q not found in session store", token)
		}
	}

	t.Run("malformed store returns read error", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/sessions.json"
		if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		broken := NewAuthService("secret", store.NewJSONStore[model.CCUsers](dir+"/users.json"), store.NewJSONStore[model.RefreshSessions](path))
		if _, err := broken.CreateRefreshSession("u1"); err == nil {
			t.Fatal("expected malformed session store error")
		}
	})
}

func TestAuthService_RevokeRefreshSession_Concurrent(t *testing.T) {
	svc, _, _ := newAtomicAuthService(t)
	token, err := svc.CreateRefreshSession("u1")
	if err != nil {
		t.Fatalf("CreateRefreshSession: %v", err)
	}
	errs := runConcurrent(t, 50, func(int) error { return svc.RevokeRefreshSession(token) })
	assertNoError(t, errs)
	if _, err := svc.ValidateRefreshSession(token); err == nil {
		t.Fatal("session remains valid after concurrent revocation")
	}
}

func TestAuthService_RevokeUserSessions_Concurrent(t *testing.T) {
	svc, _, sessionStore := newAtomicAuthService(t)
	for i := 0; i < 20; i++ {
		if _, err := svc.CreateRefreshSession("u1"); err != nil {
			t.Fatalf("CreateRefreshSession: %v", err)
		}
	}
	other, err := svc.CreateRefreshSession("u2")
	if err != nil {
		t.Fatalf("CreateRefreshSession(other): %v", err)
	}
	errs := runConcurrent(t, 20, func(int) error { return svc.RevokeUserSessions("u1") })
	assertNoError(t, errs)
	sessions, err := sessionStore.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	for _, session := range sessions.Sessions {
		if session.UserID == "u1" && !session.Revoked {
			t.Fatalf("session %s was not revoked", session.ID)
		}
		if session.ID == other && session.Revoked {
			t.Fatal("unrelated session was revoked")
		}
	}
}

func TestAuthService_PruneSessions_Concurrent(t *testing.T) {
	svc, _, sessionStore := newAtomicAuthService(t)
	now := time.Now()
	initial := model.RefreshSessions{Sessions: []model.RefreshSession{
		{ID: "old-revoked", UserID: "u1", Revoked: true, ExpiresAt: now.Add(-48 * time.Hour).Unix()},
		{ID: "old-expired", UserID: "u1", ExpiresAt: now.Add(-48 * time.Hour).Unix()},
		{ID: "recent-revoked", UserID: "u1", Revoked: true, ExpiresAt: now.Add(-time.Hour).Unix()},
		{ID: "live", UserID: "u2", ExpiresAt: now.Add(time.Hour).Unix()},
	}}
	if err := sessionStore.Write(initial); err != nil {
		t.Fatalf("Write: %v", err)
	}
	errs := runConcurrent(t, 20, func(int) error { return svc.PruneSessions() })
	assertNoError(t, errs)
	sessions, err := sessionStore.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(sessions.Sessions) != 2 || sessions.Sessions[0].ID != "recent-revoked" || sessions.Sessions[1].ID != "live" {
		t.Fatalf("unexpected pruned sessions: %+v", sessions.Sessions)
	}

	t.Run("persistence error is observable", func(t *testing.T) {
		dir := t.TempDir()
		broken := NewAuthService("secret", store.NewJSONStore[model.CCUsers](dir+"/users.json"), store.NewJSONStore[model.RefreshSessions](dir))
		if err := broken.PruneSessions(); err == nil {
			t.Fatal("expected session-store read error")
		}
	})
}

func TestAuthService_BootstrapFirstAdmin_Concurrent(t *testing.T) {
	svc, userStore, _ := newAtomicAuthService(t)
	errs := runConcurrent(t, 3, func(i int) error {
		return svc.BootstrapFirstAdmin(&model.CCUser{ID: fmt.Sprintf("u-%d", i), Username: fmt.Sprintf("admin-%d", i), PasswordHash: "hash", Role: model.RoleViewer})
	})
	succeeded, configured := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrAlreadyConfigured):
			configured++
		default:
			t.Fatalf("unexpected bootstrap error: %v", err)
		}
	}
	if succeeded != 1 || configured != 2 {
		t.Fatalf("bootstrap outcomes: succeeded=%d configured=%d", succeeded, configured)
	}
	users, err := userStore.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(users.Users) != 1 || users.Users[0].Role != model.RoleAdmin {
		t.Fatalf("first user was not uniquely persisted as admin: %+v", users.Users)
	}
}

func TestAuthService_CreateUser_Concurrent_DuplicateUsername(t *testing.T) {
	svc, userStore, _ := newAtomicAuthService(t)
	errs := runConcurrent(t, 3, func(i int) error {
		return svc.CreateUser(&model.CCUser{ID: fmt.Sprintf("u-%d", i), Username: "duplicate", Role: model.RoleViewer})
	})
	succeeded, duplicates := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrUsernameExists):
			duplicates++
		default:
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	if succeeded != 1 || duplicates != 2 {
		t.Fatalf("create outcomes: succeeded=%d duplicates=%d", succeeded, duplicates)
	}
	users, err := userStore.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(users.Users) != 1 || users.Users[0].Username != "duplicate" {
		t.Fatalf("duplicate username persisted more than once: %+v", users.Users)
	}
}
