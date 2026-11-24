package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/pquerna/otp/totp"
)

func loginFrom(h *AuthHandler, addr, password string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(LoginRequest{Username: "admin", Password: password}) // #nosec G117 -- test fixture
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.RemoteAddr = addr + ":40000"
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	return rec
}

// #1021 acceptance: six failures from one address do not block a login for
// the same user from another address.
func TestLogin_FailuresFromOneAddressDoNotLockOutAnother(t *testing.T) {
	h, _ := setupAuthTest(t)
	for i := 0; i < 6; i++ {
		if rec := loginFrom(h, "203.0.113.1", "wrong"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: want 401, got %d", i+1, rec.Code)
		}
	}
	if rec := loginFrom(h, "203.0.113.1", "password123"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("failing address must stay locked, got %d", rec.Code)
	}
	if rec := loginFrom(h, "203.0.113.2", "password123"); rec.Code != http.StatusOK {
		t.Fatalf("another address must log in, got %d (%s)", rec.Code, rec.Body.String())
	}
}

// The slower global limit still stops a password spray from many addresses.
func TestLogin_GlobalPerUserLimitStopsDistributedGuessing(t *testing.T) {
	h, _ := setupAuthTest(t)
	for i := 0; i < middleware.UserGlobalMaxAttempts; i++ {
		addr := "198.51.100." + strconv.Itoa(i/5+1) // five failures per address, under its own limit
		if rec := loginFrom(h, addr, "wrong"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: want 401, got %d", i+1, rec.Code)
		}
	}
	if rec := loginFrom(h, "192.0.2.50", "password123"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("global per-user limit must lock the account, got %d", rec.Code)
	}
}

// MFA verify shares the auth buckets, so it must follow the same per-address
// shape: wrong codes from one address do not block the user elsewhere.
func TestMfaVerify_WrongCodesFromOneAddressDoNotLockOutAnother(t *testing.T) {
	h, _, mfaSvc, uid := setupMfaHandlerTest(t)
	limiter := middleware.NewRateLimiter(t.TempDir())
	h.SetRateLimiter(limiter)
	secret, _, _ := mfaSvc.BeginEnrollment(uid)
	code, _ := totp.GenerateCode(secret, time.Now())
	if _, err := mfaSvc.ConfirmEnrollment(uid, code); err != nil {
		t.Fatalf("ConfirmEnrollment: %v", err)
	}

	verify := func(addr, tok, code string) int {
		body, _ := json.Marshal(model.MfaVerifyRequest{MfaToken: tok, Code: code})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/mfa/verify", bytes.NewReader(body))
		req.RemoteAddr = addr + ":40000"
		rec := httptest.NewRecorder()
		h.Verify(rec, req)
		return rec.Code
	}

	for i := 0; i < 6; i++ {
		tok, _ := mfaSvc.IssueMfaToken(uid) // tokens are single-use
		if got := verify("203.0.113.1", tok, "000000"); got != http.StatusUnauthorized {
			t.Fatalf("attempt %d: want 401, got %d", i+1, got)
		}
	}
	if blocked, _ := limiter.CheckAuthUser("admin", "203.0.113.1"); !blocked {
		t.Fatal("wrong codes must lock the failing address for the user")
	}
	tok, _ := mfaSvc.IssueMfaToken(uid)
	fresh, _ := totp.GenerateCode(secret, time.Now())
	if got := verify("203.0.113.2", tok, fresh); got != http.StatusOK {
		t.Fatalf("another address must verify, got %d", got)
	}
}
