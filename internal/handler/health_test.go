package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
	"github.com/go-chi/chi/v5"
)

func TestHealthz_Returns200(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.Len() != 0 {
		t.Errorf("body should be empty, got %q", w.Body.String())
	}
}

func TestAuthStatus_NoAuthRequired_Field(t *testing.T) {
	tempDir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](tempDir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](tempDir + "/sessions.json")
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)
	handler := NewAuthHandler(authSvc)

	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	w := httptest.NewRecorder()
	handler.GetStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Data struct {
			Initialized  bool  `json:"initialized"`
			AuthRequired *bool `json:"authRequired"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Data.Initialized != false {
		t.Error("initialized should be false when no users exist")
	}
	if resp.Data.AuthRequired != nil {
		t.Error("authRequired should not be present in response")
	}
}

func TestAuthStatus_InitializedWhenUsersExist(t *testing.T) {
	tempDir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](tempDir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](tempDir + "/sessions.json")
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)

	user := &model.CCUser{
		ID:           "test-id",
		Username:     "admin",
		PasswordHash: "hash",
		Role:         model.RoleAdmin,
		CreatedAt:    "2024-01-01T00:00:00Z",
		UpdatedAt:    "2024-01-01T00:00:00Z",
	}
	_ = authSvc.CreateUser(user)

	handler := NewAuthHandler(authSvc)
	req := httptest.NewRequest("GET", "/api/auth/status", nil)
	w := httptest.NewRecorder()
	handler.GetStatus(w, req)

	var resp struct {
		Data struct {
			Initialized bool `json:"initialized"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !resp.Data.Initialized {
		t.Error("initialized should be true when users exist")
	}
}
