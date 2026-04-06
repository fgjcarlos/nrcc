package service

import (
	"database/sql"
	"strings"
	"testing"
)

func TestAuthServiceBootstrapStoresHashAndAudit(t *testing.T) {
	t.Parallel()

	service := newTestAuthService(t)

	user, token, err := service.RegisterInitial("alice", "password123")
	if err != nil {
		t.Fatalf("RegisterInitial() error = %v", err)
	}
	if user == nil || token == "" {
		t.Fatalf("RegisterInitial() user/token = %#v %q", user, token)
	}

	record, err := service.findUserByUsername("alice")
	if err != nil {
		t.Fatalf("findUserByUsername() error = %v", err)
	}
	if record == nil {
		t.Fatal("findUserByUsername() = nil")
	}
	if record.PasswordHash == "password123" {
		t.Fatal("password was stored in plain text")
	}

	var eventType string
	if err := service.db.QueryRow(`SELECT event_type FROM audit_logs ORDER BY id DESC LIMIT 1`).Scan(&eventType); err != nil {
		t.Fatalf("audit log query error = %v", err)
	}
	if eventType != "auth.bootstrap" {
		t.Fatalf("audit event = %q, want %q", eventType, "auth.bootstrap")
	}

	if _, _, err := service.RegisterInitial("bob", "password123"); err == nil {
		t.Fatal("second RegisterInitial() error = nil, want rejection")
	}
}

func TestAuthServiceLoginRateLimitAndClearAfterSuccess(t *testing.T) {
	t.Parallel()

	service := newTestAuthService(t)
	if _, _, err := service.RegisterInitial("alice", "password123"); err != nil {
		t.Fatalf("RegisterInitial() error = %v", err)
	}

	for i := 0; i < 5; i++ {
		if _, _, err := service.Login("alice", "wrongpass", "127.0.0.1:1234"); err == nil || !strings.Contains(err.Error(), "invalid username or password") {
			t.Fatalf("Login() failure #%d error = %v, want invalid credentials", i+1, err)
		}
	}

	if _, _, err := service.Login("alice", "wrongpass", "127.0.0.1:1234"); err == nil || !strings.Contains(err.Error(), "too many login attempts") {
		t.Fatalf("rate-limited Login() error = %v, want rate limit", err)
	}

	service.clearLoginFailures("alice", "127.0.0.1:1234")
	if _, _, err := service.Login("alice", "password123", "127.0.0.1:1234"); err != nil {
		t.Fatalf("successful Login() error = %v", err)
	}

	if _, _, err := service.Login("alice", "wrongpass", "127.0.0.1:1234"); err == nil || !strings.Contains(err.Error(), "invalid username or password") {
		t.Fatalf("Login() after clear error = %v, want invalid credentials", err)
	}
}

func TestAuthServiceVerifyAndRevokeToken(t *testing.T) {
	t.Parallel()

	service := newTestAuthService(t)
	_, token, err := service.RegisterInitial("alice", "password123")
	if err != nil {
		t.Fatalf("RegisterInitial() error = %v", err)
	}

	claims, err := service.VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken() error = %v", err)
	}
	if claims.Sub == "" || claims.SID == "" {
		t.Fatalf("VerifyToken() claims = %+v", claims)
	}

	if err := service.RevokeToken(token); err != nil {
		t.Fatalf("RevokeToken() error = %v", err)
	}
	if _, err := service.VerifyToken(token); err == nil {
		t.Fatal("VerifyToken() after revoke error = nil, want invalid session")
	}
}

func TestClientIP(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "host and port", in: "127.0.0.1:1880", want: "127.0.0.1"},
		{name: "bare ip", in: "10.0.0.8", want: "10.0.0.8"},
		{name: "empty", in: "", want: "unknown"},
	}

	for _, tc := range cases {
		if got := clientIP(tc.in); got != tc.want {
			t.Fatalf("%s: clientIP(%q) = %q, want %q", tc.name, tc.in, got, tc.want)
		}
	}
}

func newTestAuthService(t *testing.T) *AuthService {
	t.Helper()

	service, err := NewAuthService(t.TempDir())
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}
	t.Cleanup(func() {
		if err := service.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return service
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatalf("countRows(%s) error = %v", table, err)
	}
	return count
}
