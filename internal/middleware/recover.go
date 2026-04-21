package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"nrcc/internal/model"
)

// Recoverer converts panics into the standard JSON error envelope.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				reqID := GetRequestID(r.Context())
				slog.ErrorContext(r.Context(), "panic recovered",
					slog.Any("panic", recovered),
					slog.String("request_id", reqID),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
					slog.String("stack", string(debug.Stack())),
				)

				w.Header().Set("Content-Type", "application/json")
				if reqID != "" {
					w.Header().Set("X-Request-Id", reqID)
				}
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(model.APIResponse[any]{
					Success: false,
					Error: &model.APIError{
						Code:      "INTERNAL_SERVER_ERROR",
						Message:   "internal server error",
						RequestID: reqID,
					},
					RequestID: reqID,
					Timestamp: time.Now().UTC(),
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}
