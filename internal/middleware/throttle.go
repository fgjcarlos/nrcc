package middleware

import (
	"net/http"
	"sync"
	"time"
)

// throttle is a fixed-window per-IP counter. It is deliberately NOT the
// RateLimiter used for credentials: that one is a 6-attempt/15-minute lockout
// meant to punish brute force, which would break normal navigation. This one
// only sheds abusive volume and forgets everything on restart, so it is kept
// in memory with no persistence.
type throttle struct {
	mu     sync.Mutex
	hits   map[string]*window
	limit  int
	period time.Duration
}

type window struct {
	count int
	start time.Time
}

// RateLimitIP caps requests per client IP to limit per period. Exceeding it
// returns 429 with Retry-After. Keyed via ExtractIP, so a client-supplied
// X-Forwarded-For cannot rotate buckets unless the peer is a trusted proxy.
// Closes the untrottled-endpoints half of #585 (HIGH-002).
func RateLimitIP(limit int, period time.Duration) func(http.Handler) http.Handler {
	t := &throttle{hits: make(map[string]*window), limit: limit, period: period}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if blocked, retry := t.allow(ExtractIP(r)); blocked {
				RespondTooManyRequests(w, retry)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (t *throttle) allow(ip string) (blocked bool, retryAfter time.Duration) {
	now := time.Now()

	t.mu.Lock()
	defer t.mu.Unlock()

	// ponytail: O(n) sweep over active IPs once per window, cheap for a
	// single-tenant control panel. Swap for a sharded map with its own
	// janitor if this ever fronts thousands of concurrent clients.
	for key, win := range t.hits {
		if now.Sub(win.start) >= t.period {
			delete(t.hits, key)
		}
	}

	win, ok := t.hits[ip]
	if !ok {
		t.hits[ip] = &window{count: 1, start: now}
		return false, 0
	}

	win.count++
	if win.count > t.limit {
		return true, t.period - now.Sub(win.start)
	}
	return false, 0
}
