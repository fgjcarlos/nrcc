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
	Count       int       `json:"count"`
	FirstAt     time.Time `json:"firstAt"`
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

// trustedProxies holds the networks whose X-Forwarded-For header is honored.
// It is populated from NRCC_TRUSTED_PROXIES (comma-separated CIDRs or bare IPs);
// empty by default, which means X-Forwarded-For is ignored entirely.
var trustedProxies = parseTrustedProxies(os.Getenv("NRCC_TRUSTED_PROXIES"))

// ExtractIP returns the client IP used to key rate limiting. It only honors
// X-Forwarded-For when the immediate peer (RemoteAddr) is a configured trusted
// proxy; otherwise the peer address is authoritative. This stops a direct
// client from spoofing X-Forwarded-For to rotate rate-limit buckets and bypass
// login throttling.
func ExtractIP(r *http.Request) string {
	return extractIP(r, trustedProxies)
}

func extractIP(r *http.Request, trusted []*net.IPNet) string {
	peer := peerHost(r.RemoteAddr)
	if len(trusted) > 0 && ipInNets(peer, trusted) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if first := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0]); first != "" {
				return first
			}
		}
	}
	return peer
}

func peerHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

func ipInNets(ip string, nets []*net.IPNet) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(parsed) {
			return true
		}
	}
	return false
}

// parseTrustedProxies parses a comma-separated list of CIDRs or bare IPs. Bare
// IPs become host routes (/32 or /128). Invalid entries are skipped.
func parseTrustedProxies(s string) []*net.IPNet {
	var nets []*net.IPNet
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.Contains(part, "/") {
			if strings.Contains(part, ":") {
				part += "/128"
			} else {
				part += "/32"
			}
		}
		if _, n, err := net.ParseCIDR(part); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}

func RespondTooManyRequests(w http.ResponseWriter, retryAfter time.Duration) {
	w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(`{"success":false,"error":{"code":"RATE_LIMITED","message":"Too many attempts. Please try again later."}}`))
}
