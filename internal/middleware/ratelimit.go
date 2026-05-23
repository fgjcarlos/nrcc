package middleware

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	maxAttempts    = 6
	windowDuration = 15 * time.Minute
	lockoutFile    = "ratelimit.json"
)

type attempt struct {
	Count     int       `json:"count"`
	FirstAt   time.Time `json:"firstAt"`
	LockedUntil time.Time `json:"lockedUntil,omitempty"`
}

type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string]*attempt
	dataDir  string
}

func NewRateLimiter(dataDir string) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string]*attempt),
		dataDir:  dataDir,
	}
	rl.load()
	return rl
}

func (rl *RateLimiter) Check(key string) (blocked bool, retryAfter time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	a, exists := rl.attempts[key]
	if !exists {
		return false, 0
	}

	now := time.Now()

	if !a.LockedUntil.IsZero() && now.Before(a.LockedUntil) {
		return true, time.Until(a.LockedUntil)
	}

	if now.Sub(a.FirstAt) > windowDuration {
		delete(rl.attempts, key)
		return false, 0
	}

	if a.Count >= maxAttempts {
		a.LockedUntil = a.FirstAt.Add(windowDuration)
		rl.persist()
		return true, time.Until(a.LockedUntil)
	}

	return false, 0
}

func (rl *RateLimiter) Record(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	a, exists := rl.attempts[key]
	if !exists || now.Sub(a.FirstAt) > windowDuration {
		rl.attempts[key] = &attempt{Count: 1, FirstAt: now}
		rl.persist()
		return
	}

	a.Count++
	if a.Count >= maxAttempts {
		a.LockedUntil = a.FirstAt.Add(windowDuration)
	}
	rl.persist()
}

func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, key)
	rl.persist()
}

func (rl *RateLimiter) persist() {
	if rl.dataDir == "" {
		return
	}
	data, err := json.Marshal(rl.attempts)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(rl.dataDir, lockoutFile), data, 0600)
}

func (rl *RateLimiter) load() {
	if rl.dataDir == "" {
		return
	}
	data, err := os.ReadFile(filepath.Join(rl.dataDir, lockoutFile))
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &rl.attempts)
}

func ExtractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func RespondTooManyRequests(w http.ResponseWriter, retryAfter time.Duration) {
	w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(`{"success":false,"error":{"code":"RATE_LIMITED","message":"Too many attempts. Please try again later."}}`))
}
