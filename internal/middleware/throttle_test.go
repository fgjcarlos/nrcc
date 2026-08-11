package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Login and MFA verify used separate key prefixes, so one IP got 6 login tries
// plus 6 MFA tries per window — double the intended budget (#585 HIGH-001).
func TestAuthKeys_LoginAndMfaShareOneBucket(t *testing.T) {
	rl := NewRateLimiter(t.TempDir())

	for i := 0; i < maxAttempts; i++ {
		rl.Record(AuthIPKey("203.0.113.9"))
	}

	// The MFA verify path keys off the same helper, so it must already be
	// locked out — no fresh budget for the second factor.
	if blocked, _ := rl.Check(AuthIPKey("203.0.113.9")); !blocked {
		t.Fatal("MFA verify must share the login IP bucket")
	}
}

// GetUserByUsername matches exactly, so unnormalized keys handed an attacker a
// fresh 6-try budget per spelling variant (#585 HIGH-010).
func TestAuthUserKey_NormalizesCaseAndSpace(t *testing.T) {
	want := AuthUserKey("admin")
	for _, variant := range []string{"Admin", "ADMIN", "aDmIn", "  admin  ", "Admin "} {
		if got := AuthUserKey(variant); got != want {
			t.Fatalf("AuthUserKey(%q) = %q, want %q — bucket rotation still possible", variant, got, want)
		}
	}

	rl := NewRateLimiter(t.TempDir())
	for i := 0; i < maxAttempts; i++ {
		rl.Record(AuthUserKey("admin"))
	}
	if blocked, _ := rl.Check(AuthUserKey("ADMIN")); !blocked {
		t.Fatal("case variant escaped the lockout")
	}
}

// Authenticated routes had no throttling at all: a valid JWT was enough to
// hammer any endpoint (#585 HIGH-002).
func TestRateLimitIP_CapsPerIPAndIsolatesClients(t *testing.T) {
	handler := RateLimitIP(3, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func(remoteAddr string) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/system/info", nil)
		req.RemoteAddr = remoteAddr
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	for i := 1; i <= 3; i++ {
		if code := call("203.0.113.1:1234"); code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200 (under the cap)", i, code)
		}
	}
	if code := call("203.0.113.1:1234"); code != http.StatusTooManyRequests {
		t.Fatalf("4th request: got %d, want 429", code)
	}

	// A different client must not inherit the first one's exhausted budget.
	if code := call("203.0.113.2:1234"); code != http.StatusOK {
		t.Fatalf("second IP: got %d, want 200 — buckets are not per-IP", code)
	}
}

// A spoofed X-Forwarded-For must not mint a fresh bucket when no trusted proxy
// is configured — otherwise the throttle is trivially bypassed.
func TestRateLimitIP_IgnoresSpoofedXFF(t *testing.T) {
	handler := RateLimitIP(2, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	code := 0
	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/system/info", nil)
		req.RemoteAddr = "203.0.113.5:9999"
		req.Header.Set("X-Forwarded-For", "10.0.0."+string(rune('1'+i)))
		handler.ServeHTTP(rec, req)
		code = rec.Code
	}
	if code != http.StatusTooManyRequests {
		t.Fatalf("rotating X-Forwarded-For bypassed the throttle: got %d, want 429", code)
	}
}

// The window must actually expire, or a client is locked out forever.
func TestRateLimitIP_WindowExpires(t *testing.T) {
	tr := &throttle{hits: make(map[string]*window), limit: 1, period: 20 * time.Millisecond}

	if blocked, _ := tr.allow("203.0.113.7"); blocked {
		t.Fatal("first request must pass")
	}
	if blocked, _ := tr.allow("203.0.113.7"); !blocked {
		t.Fatal("second request must be blocked")
	}

	time.Sleep(30 * time.Millisecond)

	if blocked, _ := tr.allow("203.0.113.7"); blocked {
		t.Fatal("window did not expire — client locked out permanently")
	}
}
