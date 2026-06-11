package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// CtxKeyUser is the key for storing user claims in request context
type CtxKey string

const CtxKeyUser CtxKey = "user"

// Auth middleware verifies JWT tokens in Authorization header
func Auth(authSvc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := authSvc.VerifyToken(tokenStr)
			if err != nil {
				model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
				return
			}

			// Inject claims into context
			ctx := context.WithValue(r.Context(), CtxKeyUser, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext extracts claims from request context
func ClaimsFromContext(r *http.Request) *model.Claims {
	claims, ok := r.Context().Value(CtxKeyUser).(*model.Claims)
	if !ok {
		return nil
	}
	return claims
}
