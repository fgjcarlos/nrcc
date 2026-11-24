package middleware

import (
	"encoding/json"
	"fmt"
	"log"
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

	// UserGlobalMaxAttempts is the slower per-username limit summed over all
	// addresses. It stops guessing from many addresses while one address can
	// only spend its own 6 tries, so it cannot lock the account out (#1021).
	UserGlobalMaxAttempts = 30

	authUserPrefix = "auth-user:"
)

// limitFor returns the failure budget of a bucket.
func limitFor(key string) int {
	if strings.HasPrefix(key, authUserPrefix) {
		return UserGlobalMaxAttempts
	}
	return maxAttempts
}

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

// AuthIPKey and AuthUserKey build the credential-surface bucket keys. Login and
// MFA verify are two steps of the same brute-force attempt, so they must share
// one bucket per IP and per user — separate prefixes doubled the budget to 12
// tries per 15-minute window from a single IP (#585 HIGH-001).
func AuthIPKey(ip string) string { return "auth-ip:" + ip }

// AuthUserKey is the global per-user bucket (UserGlobalMaxAttempts). It
// normalizes the username so case variants (Admin, ADMIN, aDmin) collapse into
// one bucket. Lookup is exact-match, so without this an attacker got a fresh
// budget per spelling (#585 HIGH-010).
func AuthUserKey(username string) string {
	return authUserPrefix + NormalizeUsername(username)
}

// AuthUserIPKey is the per-user bucket of one address. Keying the 6-try
// lockout on the username alone let any caller lock the account out (#1021).
func AuthUserIPKey(username, ip string) string {
	return "auth-user-ip:" + NormalizeUsername(username) + "|" + ip
}

// CheckAuthUser reports whether username is locked for ip, either by that
// address's bucket or by the global per-user bucket.
func (rl *RateLimiter) CheckAuthUser(username, ip string) (blocked bool, retryAfter time.Duration) {
	if blocked, retry := rl.Check(AuthUserIPKey(username, ip)); blocked {
		return true, retry
	}
	return rl.Check(AuthUserKey(username))
}

// RecordAuthUser counts a failed credential for username from ip.
func (rl *RateLimiter) RecordAuthUser(username, ip string) {
	rl.Record(AuthUserIPKey(username, ip))
	rl.Record(AuthUserKey(username))
}

// ResetAuthUser clears the address bucket after a success. The global bucket
// keeps its count so a distributed attacker gains nothing when the user logs in.
func (rl *RateLimiter) ResetAuthUser(username, ip string) {
	rl.Reset(AuthUserIPKey(username, ip))
}

func NormalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func SetupUserKey(username string) string {
	return "setup-user:" + NormalizeUsername(username)
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

	if a.Count >= limitFor(key) {
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
	if a.Count >= limitFor(key) {
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

// warnIgnoredXFF logs once that forwarded headers arrive without any trusted
// proxy: behind a reverse proxy every client then shares the proxy's bucket.
var warnIgnoredXFF sync.Once

// ExtractIP returns the client IP used to key rate limiting. It only honors
// X-Forwarded-For when the immediate peer (RemoteAddr) is a configured trusted
// proxy; otherwise the peer address is authoritative. This stops a direct
// client from spoofing X-Forwarded-For to rotate rate-limit buckets and bypass
// login throttling.
func ExtractIP(r *http.Request) string {
	if len(trustedProxies) == 0 && r.Header.Get("X-Forwarded-For") != "" {
		warnIgnoredXFF.Do(func() {
			log.Printf("WARN: X-Forwarded-For received but NRCC_TRUSTED_PROXIES is empty; rate limits and audit use the peer address, so behind a reverse proxy all clients share one bucket. Set NRCC_TRUSTED_PROXIES to the proxy address.")
		})
	}
	return extractIP(r, trustedProxies)
}

// extractIP walks X-Forwarded-For from the right and returns the first entry
// that is not a trusted proxy. Proxies append, so entries left of that one are
// client-written and must not pick the bucket (#1021).
func extractIP(r *http.Request, trusted []*net.IPNet) string {
	client := peerHost(r.RemoteAddr)
	if len(trusted) == 0 || !ipInNets(client, trusted) {
		return client
	}
	// A proxy may append to the client's header or add its own line.
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop := strings.TrimSpace(hops[i])
		if net.ParseIP(hop) == nil {
			break // malformed: keep the last address a trusted hop vouched for
		}
		client = hop
		if !ipInNets(hop, trusted) {
			break
		}
	}
	return client
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
	_, _ = w.Write([]byte(`{"success":false,"error":{"code":"RATE_LIMITED","message":"Too many attempts. Please try again later."}}`))
}
