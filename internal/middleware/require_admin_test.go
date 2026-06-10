package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/composedof2/nrcc/internal/model"
)

func requestWithClaims(claims *model.Claims) *http.Request {
	req := httptest.NewRequest("POST", "/api/test", nil)
	if claims != nil {
		ctx := context.WithValue(req.Context(), CtxKeyUser, claims)
		req = req.WithContext(ctx)
	}
	return req
}

func TestRequireAdmin_AllowsAdmin(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := requestWithClaims(&model.Claims{UserID: "u1", Role: model.RoleAdmin})
	w := httptest.NewRecorder()

	RequireAdmin(next).ServeHTTP(w, req)

	if !called {
		t.Error("expected next handler to be called for admin")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireAdmin_RejectsViewer(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := requestWithClaims(&model.Claims{UserID: "u2", Role: model.RoleViewer})
	w := httptest.NewRecorder()

	RequireAdmin(next).ServeHTTP(w, req)

	if called {
		t.Error("viewer must NOT reach the protected handler")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for viewer, got %d", w.Code)
	}
}

func TestRequireAdmin_RejectsMissingClaims(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := requestWithClaims(nil)
	w := httptest.NewRecorder()

	RequireAdmin(next).ServeHTTP(w, req)

	if called {
		t.Error("request without claims must NOT reach the protected handler")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when claims are missing, got %d", w.Code)
	}
}
