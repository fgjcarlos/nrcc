package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/service"
)

type contextKey string

const authClaimsContextKey contextKey = "authClaims"

func RequireAuth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(service.SessionCookieName)
			if err != nil || cookie.Value == "" {
				writeAuthError(w, r, http.StatusUnauthorized, "AUTH_REQUIRED", "authentication required")
				return
			}

			claims, err := authService.VerifyToken(cookie.Value)
			if err != nil {
				writeAuthError(w, r, http.StatusUnauthorized, "AUTH_INVALID", "invalid session")
				return
			}

			ctx := context.WithValue(r.Context(), authClaimsContextKey, *claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireCSRF(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !requiresCSRFAuth(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(service.SessionCookieName)
			if err != nil || cookie.Value == "" {
				writeAuthError(w, r, http.StatusUnauthorized, "AUTH_REQUIRED", "authentication required")
				return
			}

			token := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
			if token == "" || !authService.VerifyCSRF(cookie.Value, token) {
				writeAuthError(w, r, http.StatusForbidden, "CSRF_INVALID", "invalid csrf token")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireRole(role model.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := AuthClaimsFromContext(r.Context())
			if !ok {
				writeAuthError(w, r, http.StatusUnauthorized, "AUTH_REQUIRED", "authentication required")
				return
			}
			if claims.Role != role {
				writeAuthError(w, r, http.StatusForbidden, "AUTH_FORBIDDEN", "administrator access required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func AuthClaimsFromContext(ctx context.Context) (model.SessionClaims, bool) {
	claims, ok := ctx.Value(authClaimsContextKey).(model.SessionClaims)
	return claims, ok
}

func writeAuthError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	reqID := GetRequestID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.APIResponse[any]{
		Success: false,
		Error: &model.APIError{
			Code:      code,
			Message:   message,
			RequestID: reqID,
		},
		RequestID: reqID,
		Timestamp: time.Now().UTC(),
	})
}

func requiresCSRFAuth(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
