package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// Logger emits one structured completion record for each HTTP request.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			r, metadata := ensureRequestMetadata(r)
			w.Header().Set(requestIDHeader, metadata.requestID)
			wrapped := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			status := http.StatusOK
			defer func() {
				if recovered := recover(); recovered != nil {
					status = http.StatusInternalServerError
					logRequestCompletion(logger, r, metadata, status, time.Since(start))
					panic(recovered)
				}
				if wrapped.Status() != 0 {
					status = wrapped.Status()
				}
				logRequestCompletion(logger, r, metadata, status, time.Since(start))
			}()

			next.ServeHTTP(wrapped, r)
		})
	}
}

func logRequestCompletion(logger *slog.Logger, r *http.Request, metadata *requestMetadata, status int, duration time.Duration) {
	attrs := []any{
		"method", r.Method,
		"path", r.URL.Path,
		"status", status,
		"duration_ms", duration.Milliseconds(),
		"request_id", metadata.requestID,
	}
	if metadata.userID != "" {
		attrs = append(attrs, "user_id", metadata.userID)
	}
	logger.InfoContext(r.Context(), "http_request_completed", attrs...)
}
