package middleware

import (
	"net/http"
	"time"

	"github.com/composedof2/nrcc/internal/ui"
)

// Logger middleware logs HTTP requests with method, path, status, and duration
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrappedW := &statusCapturingWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrappedW, r)

		duration := time.Since(start)
		ui.HTTPLog(r.Method, r.URL.Path, wrappedW.statusCode, duration)
	})
}

// statusCapturingWriter wraps http.ResponseWriter to capture status code
type statusCapturingWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *statusCapturingWriter) WriteHeader(status int) {
	if !w.written {
		w.statusCode = status
		w.written = true
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *statusCapturingWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.statusCode = http.StatusOK
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}
