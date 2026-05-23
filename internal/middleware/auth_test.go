package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
	"github.com/composedof2/nrcc/internal/store"
)

func TestAuth_MissingAuthorizationHeader(t *testing.T) {
	tempDir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](tempDir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](tempDir + "/sessions.json")
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)

	authMiddleware := Auth(authSvc)

	// Handler that should not be called
	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	wrapped := authMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/api/test", nil)
	// No Authorization header
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if nextCalled {
		t.Error("Next handler should not be called without auth header")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuth_MissingBearerPrefix(t *testing.T) {
	tempDir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](tempDir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](tempDir + "/sessions.json")
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)

	authMiddleware := Auth(authSvc)

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	wrapped := authMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "InvalidTokenWithoutBearer")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if nextCalled {
		t.Error("Next handler should not be called without Bearer prefix")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	tempDir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](tempDir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](tempDir + "/sessions.json")
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)

	authMiddleware := Auth(authSvc)

	nextCalled := false
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	wrapped := authMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if nextCalled {
		t.Error("Next handler should not be called with invalid token")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	tempDir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](tempDir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](tempDir + "/sessions.json")
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)

	// Create a test user
	user := &model.CCUser{
		ID:           "test-id",
		Username:     "admin",
		PasswordHash: "hash",
		Role:         model.RoleAdmin,
		CreatedAt:    time.Now().Format(time.RFC3339),
		UpdatedAt:    time.Now().Format(time.RFC3339),
	}

	token, err := authSvc.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	authMiddleware := Auth(authSvc)

	nextCalled := false
	capturedClaims := (*model.Claims)(nil)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		capturedClaims = ClaimsFromContext(r)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := authMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if !nextCalled {
		t.Error("Next handler should be called with valid token")
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if capturedClaims == nil {
		t.Error("Claims should be injected into context")
	}
	if capturedClaims != nil && capturedClaims.Username != "admin" {
		t.Errorf("Expected username 'admin', got %s", capturedClaims.Username)
	}
}

func TestClaimsFromContext_WithValidClaims(t *testing.T) {
	claims := &model.Claims{
		Username:  "testuser",
		Role:      model.RoleAdmin,
		UserID:    "test-id",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	ctx := context.WithValue(context.Background(), CtxKeyUser, claims)
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(ctx)

	retrieved := ClaimsFromContext(req)

	if retrieved == nil {
		t.Error("ClaimsFromContext should not return nil")
	}
	if retrieved.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", retrieved.Username)
	}
	if retrieved.Role != model.RoleAdmin {
		t.Errorf("Expected role RoleAdmin, got %s", retrieved.Role)
	}
}

func TestClaimsFromContext_WithNoClaims(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(ctx)

	retrieved := ClaimsFromContext(req)

	if retrieved != nil {
		t.Error("ClaimsFromContext should return nil when no claims in context")
	}
}

func TestClaimsFromContext_WithWrongType(t *testing.T) {
	// Put wrong type in context
	ctx := context.WithValue(context.Background(), CtxKeyUser, "not a claims struct")
	req := httptest.NewRequest("GET", "/api/test", nil)
	req = req.WithContext(ctx)

	retrieved := ClaimsFromContext(req)

	if retrieved != nil {
		t.Error("ClaimsFromContext should return nil when value is wrong type")
	}
}

func TestAuth_InjectsClaimsIntoContext(t *testing.T) {
	tempDir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](tempDir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](tempDir + "/sessions.json")
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)

	user := &model.CCUser{
		ID:           "user-id",
		Username:     "testuser",
		PasswordHash: "hash",
		Role:         model.RoleViewer,
		CreatedAt:    time.Now().Format(time.RFC3339),
		UpdatedAt:    time.Now().Format(time.RFC3339),
	}

	token, err := authSvc.GenerateToken(user)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	authMiddleware := Auth(authSvc)

	var capturedRole model.UserRole
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r)
		if claims != nil {
			capturedRole = claims.Role
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := authMiddleware(nextHandler)
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if capturedRole != model.RoleViewer {
		t.Errorf("Expected RoleViewer in context, got %s", capturedRole)
	}
}

func TestAuth_MultipleRequests_IndependentContexts(t *testing.T) {
	tempDir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](tempDir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](tempDir + "/sessions.json")
	authSvc := service.NewAuthService("test-secret", userStore, sessionStore)

	user1 := &model.CCUser{
		ID:           "id1",
		Username:     "user1",
		PasswordHash: "hash1",
		Role:         model.RoleAdmin,
		CreatedAt:    time.Now().Format(time.RFC3339),
		UpdatedAt:    time.Now().Format(time.RFC3339),
	}

	user2 := &model.CCUser{
		ID:           "id2",
		Username:     "user2",
		PasswordHash: "hash2",
		Role:         model.RoleViewer,
		CreatedAt:    time.Now().Format(time.RFC3339),
		UpdatedAt:    time.Now().Format(time.RFC3339),
	}

	token1, _ := authSvc.GenerateToken(user1)
	token2, _ := authSvc.GenerateToken(user2)

	authMiddleware := Auth(authSvc)

	usernames := []string{}
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r)
		if claims != nil {
			usernames = append(usernames, claims.Username)
		}
		w.WriteHeader(http.StatusOK)
	})

	// First request
	wrapped := authMiddleware(nextHandler)
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	req1.Header.Set("Authorization", "Bearer "+token1)
	w1 := httptest.NewRecorder()
	wrapped.ServeHTTP(w1, req1)

	// Second request
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)
	w2 := httptest.NewRecorder()
	wrapped.ServeHTTP(w2, req2)

	if len(usernames) != 2 {
		t.Errorf("Expected 2 requests processed, got %d", len(usernames))
	}
	if len(usernames) >= 2 {
		if usernames[0] != "user1" || usernames[1] != "user2" {
			t.Errorf("Expected users user1 and user2, got %v", usernames)
		}
	}
}
