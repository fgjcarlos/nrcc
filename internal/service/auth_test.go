package service

import (
	"database/sql"
	"strings"
	"testing"

	"nrcc/internal/model"
)

func TestAuthServiceBootstrapStoresHashAndAudit(t *testing.T) {
	t.Parallel()

	service := newTestAuthService(t)

	user, token, err := service.RegisterInitial("alice", "Alice2025!sec")
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
	if record.PasswordHash == "Alice2025!sec" {
		t.Fatal("password was stored in plain text")
	}

	var eventType string
	if err := service.db.QueryRow(`SELECT event_type FROM audit_logs ORDER BY id DESC LIMIT 1`).Scan(&eventType); err != nil {
		t.Fatalf("audit log query error = %v", err)
	}
	if eventType != "auth.bootstrap" {
		t.Fatalf("audit event = %q, want %q", eventType, "auth.bootstrap")
	}

	if _, _, err := service.RegisterInitial("bob", "Alice2025!sec"); err == nil {
		t.Fatal("second RegisterInitial() error = nil, want rejection")
	}
}

func TestAuthServiceLoginRateLimitAndClearAfterSuccess(t *testing.T) {
	t.Parallel()

	service := newTestAuthService(t)
	if _, _, err := service.RegisterInitial("alice", "Alice2025!sec"); err != nil {
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
	if _, _, err := service.Login("alice", "Alice2025!sec", "127.0.0.1:1234"); err != nil {
		t.Fatalf("successful Login() error = %v", err)
	}

	if _, _, err := service.Login("alice", "wrongpass", "127.0.0.1:1234"); err == nil || !strings.Contains(err.Error(), "invalid username or password") {
		t.Fatalf("Login() after clear error = %v, want invalid credentials", err)
	}
}

func TestAuthServiceVerifyAndRevokeToken(t *testing.T) {
	t.Parallel()

	service := newTestAuthService(t)
	_, token, err := service.RegisterInitial("alice", "Alice2025!sec")
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

func TestAuthServiceUserManagementAndSafeguards(t *testing.T) {
	t.Parallel()

	service := newTestAuthService(t)
	admin, _, err := service.RegisterInitial("alice", "Alice2025!sec")
	if err != nil {
		t.Fatalf("RegisterInitial() error = %v", err)
	}

	created, err := service.CreateUser("operator1", "Operator2025!pass", model.RoleOperator, admin.Username)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if created.Role != model.RoleOperator {
		t.Fatalf("CreateUser() role = %q, want %q", created.Role, model.RoleOperator)
	}

	users, err := service.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("ListUsers() len = %d, want 2", len(users))
	}

	adminRecord, err := service.findUserByUsername("alice")
	if err != nil {
		t.Fatalf("findUserByUsername(admin) error = %v", err)
	}
	operatorRecord, err := service.findUserByUsername("operator1")
	if err != nil {
		t.Fatalf("findUserByUsername(operator) error = %v", err)
	}

	actor := model.SessionClaims{Sub: adminRecord.ID, Username: adminRecord.Username, Role: adminRecord.Role}
	if _, err := service.UpdateUserRole(adminRecord.ID, model.RoleOperator, actor); err == nil {
		t.Fatal("UpdateUserRole(last admin) error = nil, want safeguard")
	} else if actionErr, ok := IsUserActionError(err); !ok || actionErr.Code != "LAST_ADMIN_REQUIRED" {
		t.Fatalf("UpdateUserRole(last admin) error = %v, want LAST_ADMIN_REQUIRED", err)
	}

	if _, err := service.UpdateUserRole(operatorRecord.ID, model.RoleViewer, actor); err != nil {
		t.Fatalf("UpdateUserRole(operator) error = %v", err)
	}

	if _, _, err := service.Login("operator1", "Operator2025!pass", "127.0.0.1:1234"); err != nil {
		t.Fatalf("Login(operator) error = %v", err)
	}
	if _, err := service.ResetUserPassword(operatorRecord.ID, "Changed2025!pass", admin.Username); err != nil {
		t.Fatalf("ResetUserPassword() error = %v", err)
	}
	if _, _, err := service.Login("operator1", "Operator2025!pass", "127.0.0.1:1234"); err == nil {
		t.Fatal("Login(old password) error = nil, want failure")
	}
	if _, _, err := service.Login("operator1", "Changed2025!pass", "127.0.0.1:1234"); err != nil {
		t.Fatalf("Login(new password) error = %v", err)
	}

	if err := service.DeleteUser(adminRecord.ID, actor); err == nil {
		t.Fatal("DeleteUser(last admin) error = nil, want safeguard")
	} else if actionErr, ok := IsUserActionError(err); !ok || actionErr.Code != "LAST_ADMIN_REQUIRED" {
		t.Fatalf("DeleteUser(last admin) error = %v, want LAST_ADMIN_REQUIRED", err)
	}

	secondAdmin, err := service.CreateUser("admin2", "Admin2025!pass", model.RoleAdmin, admin.Username)
	if err != nil {
		t.Fatalf("CreateUser(second admin) error = %v", err)
	}
	if err := service.DeleteUser(adminRecord.ID, actor); err != nil {
		t.Fatalf("DeleteUser(with another admin) error = %v", err)
	}
	if user, err := service.FindPublicUserByID(adminRecord.ID); err != nil || user != nil {
		t.Fatalf("FindPublicUserByID(deleted admin) = %#v, %v; want nil, nil", user, err)
	}
	if user, err := service.FindPublicUserByID(secondAdmin.ID); err != nil || user == nil {
		t.Fatalf("FindPublicUserByID(second admin) = %#v, %v; want user, nil", user, err)
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

// TestAuthServiceCloseStopsGoroutine verifies that Close() signals the cleanup
// goroutine to stop before closing the DB. If the goroutine were still running
// after Close() it would attempt DB writes against a closed connection, causing
// errors visible as race conditions or panics in tests.
func TestAuthServiceCloseStopsGoroutine(t *testing.T) {
	t.Parallel()

	svc, err := NewAuthService(t.TempDir())
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	// stopCh must be open before Close().
	select {
	case <-svc.stopCh:
		t.Fatal("stopCh already closed before Close()")
	default:
	}

	if err := svc.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// stopCh must be closed (readable) after Close().
	select {
	case <-svc.stopCh:
		// expected: goroutine received the signal and will exit
	default:
		t.Fatal("stopCh not closed after Close(); goroutine leak likely")
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
