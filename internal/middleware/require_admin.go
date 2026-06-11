package middleware

import (
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// RequireAdmin restricts a route to users with the admin role. It MUST be
// chained AFTER Auth (which validates the token and injects the claims): missing
// claims yield 401, a non-admin role yields 403. Use it on every state-mutating
// endpoint so viewer tokens cannot install packages, apply updates, change env
// vars, manage backups, or revert flows.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r)
		if claims == nil {
			model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
			return
		}
		if claims.Role != model.RoleAdmin {
			model.RespondError(w, http.StatusForbidden, "FORBIDDEN", "Admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
