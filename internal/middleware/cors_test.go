package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_DenyByDefault(t *testing.T) {
	handler := CORS(CORSConfig{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if acao := w.Header().Get("Access-Control-Allow-Origin"); acao != "" {
		t.Errorf("should deny unknown origins, got Access-Control-Allow-Origin = %q", acao)
	}
}

func TestCORS_AllowConfiguredOrigin(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://app.example.com"}}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "https://app.example.com")
	}
	if vary := w.Header().Get("Vary"); vary != "Origin" {
		t.Errorf("Vary = %q, want %q", vary, "Origin")
	}
}

func TestCORS_RejectUnconfiguredOrigin(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://app.example.com"}}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "https://other.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if acao := w.Header().Get("Access-Control-Allow-Origin"); acao != "" {
		t.Errorf("should reject unconfigured origin, got Access-Control-Allow-Origin = %q", acao)
	}
}

func TestCORS_UnsafeWildcard(t *testing.T) {
	cfg := CORSConfig{AllowUnsafeWildcard: true}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "https://anything.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

func TestCORS_PreflightResponse(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://app.example.com"}}
	nextCalled := false
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if nextCalled {
		t.Error("next handler should not be called for preflight")
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("preflight should set Access-Control-Allow-Methods")
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("preflight should set Access-Control-Allow-Headers")
	}
	if got := w.Header().Get("Access-Control-Max-Age"); got == "" {
		t.Error("preflight should set Access-Control-Max-Age")
	}
}

func TestCORS_PreflightDenied_WhenOriginNotAllowed(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://app.example.com"}}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if acao := w.Header().Get("Access-Control-Allow-Origin"); acao != "" {
		t.Errorf("should not set ACAO for denied preflight, got %q", acao)
	}
}

func TestCORS_NoOriginHeader_PassesThrough(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://app.example.com"}}
	called := false
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("next handler should be called when no Origin header")
	}
	if acao := w.Header().Get("Access-Control-Allow-Origin"); acao != "" {
		t.Errorf("should not set CORS headers without Origin, got %q", acao)
	}
}

func TestCORS_MultipleOrigins(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{
		"https://app.example.com",
		"https://admin.example.com",
	}}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, origin := range cfg.AllowedOrigins {
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if got := w.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("origin %s: Access-Control-Allow-Origin = %q, want %q", origin, got, origin)
		}
	}
}
