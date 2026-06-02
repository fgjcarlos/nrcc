package handler

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestIsSecureRequest verifies that isSecureRequest correctly identifies
// whether a request should use a Secure cookie based on TLS state and
// X-Forwarded-Proto header.
func TestIsSecureRequest(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func() *http.Request
		wantSecure   bool
	}{
		{
			name: "plain HTTP — no TLS, no forwarded proto",
			setupRequest: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
				// r.TLS is nil by default, no X-Forwarded-Proto header
				return r
			},
			wantSecure: false,
		},
		{
			name: "native TLS — r.TLS is set (direct HTTPS, no proxy)",
			setupRequest: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
				r.TLS = &tls.ConnectionState{}
				return r
			},
			wantSecure: true,
		},
		{
			name: "X-Forwarded-Proto: https — proxy terminates TLS",
			setupRequest: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
				r.Header.Set("X-Forwarded-Proto", "https")
				return r
			},
			wantSecure: true,
		},
		{
			name: "X-Forwarded-Proto: HTTPS — case-insensitive match",
			setupRequest: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
				r.Header.Set("X-Forwarded-Proto", "HTTPS")
				return r
			},
			wantSecure: true,
		},
		{
			name: "X-Forwarded-Proto: http — explicitly HTTP, not secure",
			setupRequest: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
				r.Header.Set("X-Forwarded-Proto", "http")
				return r
			},
			wantSecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.setupRequest()
			got := isSecureRequest(r)
			if got != tt.wantSecure {
				t.Errorf("isSecureRequest() = %v, want %v", got, tt.wantSecure)
			}
		})
	}
}

// TestSetRefreshCookie_SecureAttribute verifies that the Secure attribute on
// the nrcc_refresh cookie is set conditionally based on the request context:
//   - plain HTTP (no TLS, no X-Forwarded-Proto) → Secure == false
//   - X-Forwarded-Proto: https (TLS terminated upstream) → Secure == true
func TestSetRefreshCookie_SecureAttribute(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func() *http.Request
		wantSecure   bool
	}{
		{
			name: "plain HTTP request sets Secure=false",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
			},
			wantSecure: false,
		},
		{
			name: "X-Forwarded-Proto: https sets Secure=true",
			setupRequest: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
				r.Header.Set("X-Forwarded-Proto", "https")
				return r
			},
			wantSecure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := setupAuthTest(t)
			r := tt.setupRequest()
			w := httptest.NewRecorder()

			// Call the method under test: setRefreshCookie must accept *http.Request
			// to determine the Secure flag.
			h.setRefreshCookie(w, r, "test-user-id")

			// Parse the Set-Cookie header to inspect the Secure attribute.
			resp := w.Result()
			var refreshCookie *http.Cookie
			for _, c := range resp.Cookies() {
				if c.Name == refreshCookieName {
					refreshCookie = c
					break
				}
			}
			if refreshCookie == nil {
				t.Fatal("setRefreshCookie did not set a cookie")
			}
			if refreshCookie.Secure != tt.wantSecure {
				t.Errorf("cookie Secure = %v, want %v (Set-Cookie header: %q)",
					refreshCookie.Secure, tt.wantSecure,
					w.Header().Get("Set-Cookie"))
			}
		})
	}
}

// TestClearRefreshCookie_SecureAttribute verifies that clearRefreshCookie sets
// Secure consistently with the original cookie (so the browser matches and
// deletes it). Secure must be false on plain HTTP, true when proxied via HTTPS.
func TestClearRefreshCookie_SecureAttribute(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func() *http.Request
		wantSecure   bool
	}{
		{
			name: "plain HTTP clear sets Secure=false",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
			},
			wantSecure: false,
		},
		{
			name: "X-Forwarded-Proto: https clear sets Secure=true",
			setupRequest: func() *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
				r.Header.Set("X-Forwarded-Proto", "https")
				return r
			},
			wantSecure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.setupRequest()
			w := httptest.NewRecorder()

			// Call the function under test: clearRefreshCookie must accept
			// *http.Request to determine the Secure flag.
			clearRefreshCookie(w, r)

			resp := w.Result()
			var refreshCookie *http.Cookie
			for _, c := range resp.Cookies() {
				if c.Name == refreshCookieName {
					refreshCookie = c
					break
				}
			}
			if refreshCookie == nil {
				t.Fatal("clearRefreshCookie did not set a cookie")
			}
			if refreshCookie.Secure != tt.wantSecure {
				t.Errorf("cookie Secure = %v, want %v (Set-Cookie header: %q)",
					refreshCookie.Secure, tt.wantSecure,
					w.Header().Get("Set-Cookie"))
			}
			// Clear cookie must also have MaxAge -1 (deleted)
			if refreshCookie.MaxAge != -1 {
				t.Errorf("clear cookie MaxAge = %d, want -1", refreshCookie.MaxAge)
			}
		})
	}
}

// TestLogin_SecureAttributeConditional verifies the full login flow sets the
// Secure cookie attribute based on the request protocol, not a hardcoded value.
func TestLogin_SecureAttributeConditional(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func() *http.Request
		wantSecure   bool
	}{
		{
			name: "login over plain HTTP → cookie Secure=false",
			setupRequest: func() *http.Request {
				body := strings.NewReader(`{"username":"admin","password":"password123"}`)
				r := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
				r.Header.Set("Content-Type", "application/json")
				return r
			},
			wantSecure: false,
		},
		{
			name: "login proxied via HTTPS → cookie Secure=true",
			setupRequest: func() *http.Request {
				body := strings.NewReader(`{"username":"admin","password":"password123"}`)
				r := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("X-Forwarded-Proto", "https")
				return r
			},
			wantSecure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := setupAuthTest(t)
			r := tt.setupRequest()
			w := httptest.NewRecorder()

			h.Login(w, r)

			if w.Code != http.StatusOK {
				t.Fatalf("Login returned %d, want 200; body: %s", w.Code, w.Body.String())
			}

			var refreshCookie *http.Cookie
			for _, c := range w.Result().Cookies() {
				if c.Name == refreshCookieName {
					refreshCookie = c
					break
				}
			}
			if refreshCookie == nil {
				t.Fatal("Login did not set a refresh cookie")
			}
			if refreshCookie.Secure != tt.wantSecure {
				t.Errorf("cookie Secure = %v, want %v", refreshCookie.Secure, tt.wantSecure)
			}
		})
	}
}
