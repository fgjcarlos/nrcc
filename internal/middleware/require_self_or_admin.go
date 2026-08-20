package middleware

import (
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// RequireSelfOrAdmin restricts a route to the target user (identified by the
// caller-supplied extractor) OR an admin. It is the self-or-admin counterpart
// to RequireAdmin. Use it on routes where an authenticated caller may operate
// on their own resource (e.g. changing their own password, disabling their own
// MFA) but the same route must reject anyone else.
//
// The extractor returns (targetUserID, ok). A target that fails the ok check
// (or is empty) yields a 403 — the route must declare where the target comes
// from, and the absence of a target is treated as an authorization failure
// rather than a fall-through to the handler.
//
// Like RequireAdmin, this MUST be chained AFTER Auth: missing claims yield
// 401. The extractor is called with the request the handler will see, so
// middleware that wraps the request (e.g. body-peeking for a body-sourced
// target) must restore r.Body before returning.
func RequireSelfOrAdmin(target func(*http.Request) (string, bool)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r)
			if claims == nil {
				model.RespondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}
			if claims.Role == model.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}
			targetID, ok := target(r)
			if !ok || targetID == "" {
				model.RespondError(w, http.StatusForbidden, "FORBIDDEN", "Target user id required")
				return
			}
			if claims.UserID != targetID {
				model.RespondError(w, http.StatusForbidden, "FORBIDDEN", "Self or admin access required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}