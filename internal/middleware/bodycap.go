package middleware

import (
	"net/http"
	"strings"
)

const (
	// DefaultBodyLimit is the maximum request body size for routes without an override.
	DefaultBodyLimit int64 = 1 << 20
	miB              int64 = 1 << 20
)

// BodyLimitConfig defines the default request body limit and exact route overrides.
type BodyLimitConfig struct {
	DefaultLimit int64
	Overrides    map[string]int64
}

// DefaultBodyLimitConfig returns the secure default request body policy.
func DefaultBodyLimitConfig() BodyLimitConfig {
	return BodyLimitConfig{
		DefaultLimit: DefaultBodyLimit,
		Overrides: map[string]int64{
			http.MethodPost + " /api/env/bulk":        5 * miB,
			http.MethodPut + " /api/env/dotenv":       5 * miB,
			http.MethodPost + " /api/settings/raw":    2 * miB,
			http.MethodPost + " /api/ai/analyze/flow": 5 * miB,
		},
	}
}

// BodyLimitMiddleware caps non-multipart request bodies using exact method and path policy keys.
func BodyLimitMiddleware(cfg BodyLimitConfig) func(http.Handler) http.Handler {
	defaultLimit := cfg.DefaultLimit
	if defaultLimit <= 0 {
		defaultLimit = DefaultBodyLimit
	}

	overrides := make(map[string]int64, len(cfg.Overrides))
	for route, limit := range cfg.Overrides {
		if limit > 0 {
			overrides[route] = limit
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
			if strings.HasPrefix(contentType, "multipart/") {
				next.ServeHTTP(w, r)
				return
			}

			limit := defaultLimit
			if override, ok := overrides[r.Method+" "+r.URL.Path]; ok {
				limit = override
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
