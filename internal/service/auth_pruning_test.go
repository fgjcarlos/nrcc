package service

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestPruneSessionsUsesFirstRevocationTime(t *testing.T) {
	for _, method := range []string{"single session", "all user sessions"} {
		t.Run(method, func(t *testing.T) {
			svc, _, _ := newAtomicAuthService(t)
			synctest.Test(t, func(t *testing.T) {
				token, err := svc.CreateRefreshSession("user")
				if err != nil {
					t.Fatal(err)
				}
				revoke := func() error { return svc.RevokeRefreshSession(token) }
				if method == "all user sessions" {
					revoke = func() error { return svc.RevokeUserSessions("user") }
				}
				if err := revoke(); err != nil {
					t.Fatal(err)
				}
				time.Sleep(24 * time.Hour) // Advances the synctest clock.
				if err := svc.PruneSessions(); err != nil {
					t.Fatal(err)
				}
				if _, err := svc.ValidateRefreshSession(token); err == nil || err.Error() != "refresh token revoked" {
					t.Fatalf("at cutoff: got %v, want revoked", err)
				}
				time.Sleep(time.Second)
				if err := revoke(); err != nil {
					t.Fatal(err)
				}
				if err := svc.PruneSessions(); err != nil {
					t.Fatal(err)
				}
				if _, err := svc.ValidateRefreshSession(token); err == nil || err.Error() != "refresh token not found" {
					t.Fatalf("past cutoff: got %v, want not found", err)
				}
			})
		})
	}
}
