package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

func TestRequireSelfOrAdmin_AllowsAdmin(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	target := func(*http.Request) (string, bool) { return "other-user", true }

	req := requestWithClaims(&model.Claims{UserID: "u1", Role: model.RoleAdmin})
	w := httptest.NewRecorder()

	RequireSelfOrAdmin(target)(next).ServeHTTP(w, req)

	if !called {
		t.Error("admin must reach the protected handler regardless of target")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRequireSelfOrAdmin_AllowsSelf(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	target := func(*http.Request) (string, bool) { return "u2", true }

	req := requestWithClaims(&model.Claims{UserID: "u2", Role: model.RoleViewer})
	w := httptest.NewRecorder()

	RequireSelfOrAdmin(target)(next).ServeHTTP(w, req)

	if !called {
		t.Error("viewer targeting self must reach the protected handler")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for self, got %d", w.Code)
	}
}

func TestRequireSelfOrAdmin_RejectsOtherUser(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	target := func(*http.Request) (string, bool) { return "u3", true }

	req := requestWithClaims(&model.Claims{UserID: "u2", Role: model.RoleViewer})
	w := httptest.NewRecorder()

	RequireSelfOrAdmin(target)(next).ServeHTTP(w, req)

	if called {
		t.Error("non-admin viewer targeting another user must NOT reach the handler")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-self non-admin, got %d", w.Code)
	}
}

func TestRequireSelfOrAdmin_RejectsMissingTarget(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	target := func(*http.Request) (string, bool) { return "", false }

	req := requestWithClaims(&model.Claims{UserID: "u2", Role: model.RoleViewer})
	w := httptest.NewRecorder()

	RequireSelfOrAdmin(target)(next).ServeHTTP(w, req)

	if called {
		t.Error("missing target must NOT reach the handler")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when target is missing, got %d", w.Code)
	}
}

func TestRequireSelfOrAdmin_RejectsMissingClaims(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	target := func(*http.Request) (string, bool) { return "u2", true }

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	ctx := context.WithValue(req.Context(), CtxKeyUser, (*model.Claims)(nil))
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	RequireSelfOrAdmin(target)(next).ServeHTTP(w, req)

	if called {
		t.Error("request without claims must NOT reach the protected handler")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when claims are missing, got %d", w.Code)
	}
}