package middleware

import (
	"net/http"
	"path"
	"strings"
)

const (
	// DefaultBodyLimit is the maximum request body size for routes without an override.
	DefaultBodyLimit int64 = 1 << 20
	miB              int64 = 1 << 20
)

// BodyLimitConfig defines the default request body limit and route overrides.
// An override key is "METHOD /path"; a key containing "*" is a path.Match
// pattern where "*" matches one path segment, checked after exact keys.
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
			// A preset preview carries the same settings.js source as a raw save.
			http.MethodPost + " /api/presets/*/preview": 2 * miB,
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
	patterns := make(map[string]int64)
	for route, limit := range cfg.Overrides {
		if limit <= 0 {
			continue
		}
		if strings.Contains(route, "*") {
			patterns[route] = limit
		} else {
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

			key := r.Method + " " + r.URL.Path
			limit := defaultLimit
			if override, ok := overrides[key]; ok {
				limit = override
			} else {
				for pattern, override := range patterns {
					if matched, _ := path.Match(pattern, key); matched {
						limit = override
						break
					}
				}
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
