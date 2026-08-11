package middleware

import (
	"testing"
)

// server.go used to build two RateLimiter instances at startup — both load an
// empty file, then each persist() serializes only its own map, so the second
// writer drops the first's buckets and the lockout silently disappears.
// Regression guard for the shared-instance wiring (#615).
func TestRateLimiter_SeparateInstancesClobberSharedFile(t *testing.T) {
	dir := t.TempDir()

	// Both constructed at startup, before any attempt is recorded.
	auth := NewRateLimiter(dir)
	mfa := NewRateLimiter(dir)

	for i := 0; i < maxAttempts; i++ {
		auth.Record("ip:10.0.0.1")
	}
	if blocked, _ := auth.Check("ip:10.0.0.1"); !blocked {
		t.Fatal("expected lockout after maxAttempts")
	}

	// The MFA limiter's write wipes the login lockout from disk.
	mfa.Record("mfa-verify-ip:10.0.0.2")

	if blocked, _ := NewRateLimiter(dir).Check("ip:10.0.0.1"); blocked {
		t.Fatal("unexpected: separate instances no longer clobber; " +
			"the shared-instance requirement in server.go may be revisitable")
	}

	// One shared instance keeps both buckets across the same sequence.
	dir2 := t.TempDir()
	shared := NewRateLimiter(dir2)
	for i := 0; i < maxAttempts; i++ {
		shared.Record("ip:10.0.0.1")
	}
	shared.Record("mfa-verify-ip:10.0.0.2")

	after := NewRateLimiter(dir2)
	if blocked, _ := after.Check("ip:10.0.0.1"); !blocked {
		t.Fatal("shared instance must keep the login lockout across an MFA write")
	}
	if _, ok := after.attempts["mfa-verify-ip:10.0.0.2"]; !ok {
		t.Fatal("shared instance must keep the MFA bucket")
	}
}
