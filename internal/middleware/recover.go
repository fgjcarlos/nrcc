package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/ui"
)

// Recoverer is a middleware that recovers from panics in downstream handlers
// and middleware. It:
//   - Logs the panic value and full stack trace server-side (never to the client).
//   - Responds with HTTP 500 and a JSON body {"error": "internal server error"}.
//   - Re-panics http.ErrAbortHandler so chi can perform its own connection
//     tear-down (not swallowed).
//
// Limitation: if a downstream handler already wrote a status code or body before
// panicking, the 500 response cannot replace it — the first WriteHeader wins, so
// the client may receive the partial response with the error JSON appended. This
// matches the behaviour of chi's own Recoverer; handlers that stream partial
// output must guard their own writes.
//
// Recoverer MUST be registered as the FIRST r.Use(...) so it wraps every
// subsequent middleware and every route handler.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// Re-panic http.ErrAbortHandler — chi/net/http use this sentinel
				// to abort the connection cleanly. Swallowing it would cause a
				// goroutine leak or unexpected behaviour.
				if rec == http.ErrAbortHandler {
					panic(rec)
				}

				// Log panic value and stack SERVER-SIDE ONLY.
				stack := debug.Stack()
				ui.Errorf("panic recovered: %v\n%s", rec, stack)

				// Respond with a generic 500 — no stack, no panic value to client.
				model.RespondJSON(w, http.StatusInternalServerError, map[string]interface{}{
					"error": "internal server error",
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}
