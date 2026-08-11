package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/service"
)

func setupRateLimitedAuthTest(t *testing.T) (*AuthHandler, *service.AuthService) {
	t.Helper()

	h, authSvc := setupAuthTest(t)
	return h, authSvc
}

func refreshRequest(token string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: token})
	}
	return req
}

func TestAuthHandler_Refresh_RateLimit_IP(t *testing.T) {
	h, _ := setupRateLimitedAuthTest(t)
	recorders := make([]*httptest.ResponseRecorder, 12)

	runConcurrentHandler(t, len(recorders), func(i int) {
		recorders[i] = httptest.NewRecorder()
		h.Refresh(recorders[i], refreshRequest("invalid-refresh-token"))
	})

	var unauthorized, rateLimited int
	for i, recorder := range recorders {
		switch recorder.Code {
		case http.StatusUnauthorized:
			unauthorized++
		case http.StatusTooManyRequests:
			rateLimited++
			if recorder.Header().Get("Retry-After") == "" {
				t.Errorf("response %d: expected Retry-After header", i)
			}
		default:
			t.Errorf("response %d: unexpected status %d", i, recorder.Code)
		}
	}

	if unauthorized != 6 {
		t.Errorf("unauthorized responses = %d; want 6", unauthorized)
	}
	if rateLimited != 6 {
		t.Errorf("rate-limited responses = %d; want 6", rateLimited)
	}
}

func TestAuthHandler_Refresh_RateLimit_ResetOnSuccess(t *testing.T) {
	h, authSvc := setupRateLimitedAuthTest(t)

	for i := 0; i < 5; i++ {
		recorder := httptest.NewRecorder()
		h.Refresh(recorder, refreshRequest("invalid-refresh-token"))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("invalid request %d: status = %d; want 401", i+1, recorder.Code)
		}
	}

	user := authSvc.GetUserByUsername("admin")
	refreshToken, err := authSvc.CreateRefreshSession(user.ID)
	if err != nil {
		t.Fatalf("CreateRefreshSession: %v", err)
	}
	success := httptest.NewRecorder()
	h.Refresh(success, refreshRequest(refreshToken))
	if success.Code != http.StatusOK {
		t.Fatalf("valid refresh status = %d; want 200: %s", success.Code, success.Body.String())
	}

	afterReset := httptest.NewRecorder()
	h.Refresh(afterReset, refreshRequest("invalid-after-success"))
	if afterReset.Code != http.StatusUnauthorized {
		t.Fatalf("post-success invalid status = %d; want 401", afterReset.Code)
	}
}

func TestAuthHandler_Refresh_NoRateLimit_OnRotationFailure(t *testing.T) {
	h, authSvc := setupRateLimitedAuthTest(t)

	for i := 0; i < 5; i++ {
		recorder := httptest.NewRecorder()
		h.Refresh(recorder, refreshRequest("invalid-refresh-token"))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("invalid request %d: status = %d; want 401", i+1, recorder.Code)
		}
	}

	user := authSvc.GetUserByUsername("admin")
	refreshToken, err := authSvc.CreateRefreshSession(user.ID)
	if err != nil {
		t.Fatalf("CreateRefreshSession: %v", err)
	}
	h.createRefreshSession = func(string) (string, error) {
		return "", errors.New("forced refresh-session creation failure")
	}

	failedRotation := httptest.NewRecorder()
	h.Refresh(failedRotation, refreshRequest(refreshToken))
	if failedRotation.Code != http.StatusInternalServerError {
		t.Fatalf("failed rotation status = %d; want 500: %s", failedRotation.Code, failedRotation.Body.String())
	}

	thresholdFailure := httptest.NewRecorder()
	h.Refresh(thresholdFailure, refreshRequest("invalid-after-failure"))
	if thresholdFailure.Code != http.StatusUnauthorized {
		t.Fatalf("threshold invalid status = %d; want 401", thresholdFailure.Code)
	}

	blocked := httptest.NewRecorder()
	h.Refresh(blocked, refreshRequest("invalid-while-blocked"))
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("post-threshold status = %d; want 429", blocked.Code)
	}
	if blocked.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header after preserved failures reach the threshold")
	}
}
