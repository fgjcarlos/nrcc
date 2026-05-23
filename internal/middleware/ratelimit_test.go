package middleware

import (
	"testing"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := NewRateLimiter(t.TempDir())

	for i := 0; i < maxAttempts-1; i++ {
		rl.Record("ip:1.2.3.4")
	}

	blocked, _ := rl.Check("ip:1.2.3.4")
	if blocked {
		t.Error("should not block under the limit")
	}
}

func TestRateLimiter_BlocksAtLimit(t *testing.T) {
	rl := NewRateLimiter(t.TempDir())

	for i := 0; i < maxAttempts; i++ {
		rl.Record("ip:1.2.3.4")
	}

	blocked, retry := rl.Check("ip:1.2.3.4")
	if !blocked {
		t.Error("should block at the limit")
	}
	if retry <= 0 {
		t.Error("retry-after should be positive")
	}
}

func TestRateLimiter_UsernameBlocking(t *testing.T) {
	rl := NewRateLimiter(t.TempDir())

	for i := 0; i < maxAttempts; i++ {
		rl.Record("user:admin")
	}

	blocked, _ := rl.Check("user:admin")
	if !blocked {
		t.Error("should block username after max attempts")
	}

	blocked, _ = rl.Check("user:other")
	if blocked {
		t.Error("should not block unrelated username")
	}
}

func TestRateLimiter_ResetClearsBlock(t *testing.T) {
	rl := NewRateLimiter(t.TempDir())

	for i := 0; i < maxAttempts; i++ {
		rl.Record("ip:1.2.3.4")
	}

	blocked, _ := rl.Check("ip:1.2.3.4")
	if !blocked {
		t.Fatal("should be blocked")
	}

	rl.Reset("ip:1.2.3.4")

	blocked, _ = rl.Check("ip:1.2.3.4")
	if blocked {
		t.Error("should be unblocked after reset")
	}
}

func TestRateLimiter_PersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	rl1 := NewRateLimiter(dir)

	for i := 0; i < maxAttempts; i++ {
		rl1.Record("ip:5.6.7.8")
	}

	rl2 := NewRateLimiter(dir)
	blocked, _ := rl2.Check("ip:5.6.7.8")
	if !blocked {
		t.Error("lockout state should persist across instances")
	}
}

func TestRateLimiter_IndependentKeys(t *testing.T) {
	rl := NewRateLimiter(t.TempDir())

	for i := 0; i < maxAttempts; i++ {
		rl.Record("ip:1.1.1.1")
	}

	blocked, _ := rl.Check("ip:1.1.1.1")
	if !blocked {
		t.Fatal("1.1.1.1 should be blocked")
	}

	blocked, _ = rl.Check("ip:2.2.2.2")
	if blocked {
		t.Error("2.2.2.2 should not be blocked")
	}
}
