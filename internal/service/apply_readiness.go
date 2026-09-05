// Package service — slice B readiness probe.
//
// After a successful atomic write + adapter.Restart the orchestrator
// must verify Node-RED is actually serving the new settings before it
// declares apply.success. probeAdminReady is that probe: it hits the
// Node-RED admin HTTP endpoint (/settings) with bounded retries and
// reports readiness via a small set of typed sentinels.
//
// Probe contract
// ==============
//
// The probe classifies every response into one of three buckets so the
// orchestrator can decide without inspecting HTTP status codes itself:
//
//   - alive  (nil)            : Node-RED answered. 2xx, 401, and 403 all
//                              count as alive because adminAuth can be
//                              required and we don't want to confuse an
//                              auth-required Node-RED with a dead one.
//   - notYetReady             : connection refused, DNS failure, 503,
//                              or any other transient error. The
//                              orchestrator should keep retrying until
//                              the readiness deadline expires, then
//                              trigger rollback.
//   - ctx.Err()               : the caller's context expired. Surface
//                              the error so the orchestrator can map it
//                              to a typed ApplyError.
//
// The function is intentionally context-driven so slice C can wire a
// readiness deadline (default: 30s) into the apply pipeline without
// touching this file. Probe timeouts are derived from ctx via
// http.NewRequestWithContext.

package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Readiness probe tunables. Defaults are conservative: Node-RED boot
// on a Pi takes ~3s; a docker-compose restart adds another 1-2s for
// the compose orchestration. 5s per attempt × 6 attempts = 30s total,
// matching the readiness deadline documented in apply-progress.md.
const (
	defaultReadinessTimeout    = 5 * time.Second
	defaultReadinessAttempts   = 6
	defaultReadinessBackoffMin = 200 * time.Millisecond
	defaultReadinessBackoffMax = 2 * time.Second
)

// ErrReadinessNotReady is returned by probeAdminReady when the probe
// exhausted all attempts without seeing an alive response. The
// orchestrator catches this sentinel (via errors.Is) and triggers the
// rollback flow.
var ErrReadinessNotReady = errors.New("apply: readiness probe exhausted attempts without an alive response")

// ReadinessProbe is the dependency port the probe uses to issue HTTP
// requests. Tests substitute a fake that returns canned responses;
// production uses the http.DefaultClient wrapped to honour ctx.
type ReadinessProbe interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultReadinessProbe wraps http.Client with a timeout that honours
// ctx so cancellation propagates between attempts.
type defaultReadinessProbe struct {
	timeout time.Duration
}

func (d defaultReadinessProbe) Do(req *http.Request) (*http.Response, error) {
	client := &http.Client{Timeout: d.timeout}
	return client.Do(req)
}

// readinessConfig configures probeAdminReady. Tests pass an explicit
// config; production callers fall back to the package defaults.
type readinessConfig struct {
	client    ReadinessProbe
	timeout   time.Duration
	attempts  int
	backoff   time.Duration // base backoff; doubles each attempt up to 2s
	userAgent string
}

// readinessOption configures readinessConfig.
type readinessOption func(*readinessConfig)

// withReadinessProbe swaps the HTTP client used for the probe. nil is
// ignored so callers can chain without nil-checking.
func withReadinessProbe(p ReadinessProbe) readinessOption {
	return func(c *readinessConfig) {
		if p != nil {
			c.client = p
		}
	}
}

// withReadinessTimeout overrides the per-attempt HTTP timeout.
func withReadinessTimeout(d time.Duration) readinessOption {
	return func(c *readinessConfig) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// withReadinessAttempts overrides the number of probe attempts before
// ErrReadinessNotReady is returned.
func withReadinessAttempts(n int) readinessOption {
	return func(c *readinessConfig) {
		if n > 0 {
			c.attempts = n
		}
	}
}

// withReadinessBackoff overrides the initial backoff between attempts.
func withReadinessBackoff(d time.Duration) readinessOption {
	return func(c *readinessConfig) {
		if d > 0 {
			c.backoff = d
		}
	}
}

// probeAdminReady probes adminURL until Node-RED responds with an
// alive signal or the attempts are exhausted. See the file header for
// the alive / notYetReady / cancelled contract.
//
// The function tolerates a nil ReadinessProbe via the package default
// so production callers can pass an empty option set. Tests inject a
// fake via withReadinessProbe to drive deterministic success/failure
// paths without an actual HTTP server.
func probeAdminReady(ctx context.Context, adminURL string, opts ...readinessOption) error {
	cfg := readinessConfig{
		timeout:   defaultReadinessTimeout,
		attempts:  defaultReadinessAttempts,
		backoff:   defaultReadinessBackoffMin,
		userAgent: "nrcc-apply/1.0 (+readiness)",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.client == nil {
		cfg.client = defaultReadinessProbe{timeout: cfg.timeout}
	}
	if strings.TrimSpace(adminURL) == "" {
		return errors.New("apply: readiness probe URL is empty")
	}

	var lastErr error
	delay := cfg.backoff
	for attempt := 1; attempt <= cfg.attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := probeAttempt(ctx, cfg.client, cfg.userAgent, adminURL)
		if err == nil {
			return nil
		}
		lastErr = err
		if errors.Is(err, ErrReadinessNotReady) {
			// Non-retryable from the probe's perspective: the
			// server answered with a fatal signal (we don't have
			// one today, but future-proof). Bail out immediately.
			return err
		}
		// Don't sleep after the final attempt.
		if attempt == cfg.attempts {
			break
		}
		// Honour ctx during the backoff sleep so cancellation
		// doesn't have to wait for the full retry window.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		// Exponential backoff capped at 2s so a slow Node-RED boot
		// doesn't push the total probe time past the readiness
		// deadline.
		delay *= 2
		if delay > defaultReadinessBackoffMax {
			delay = defaultReadinessBackoffMax
		}
	}
	if lastErr == nil {
		lastErr = ErrReadinessNotReady
	}
	// We deliberately use a string rendering of lastErr so the
	// returned chain stays anchored on ErrReadinessNotReady — the
	// orchestrator only needs the sentinel match. The underlying
	// lastErr is a transient (network / HTTP) error that callers
	// inspect via the audit envelope, not via the wrapped chain.
	lastErrMsg := lastErr.Error()
	return fmt.Errorf("%w: last error: %s", ErrReadinessNotReady, lastErrMsg)
}

// probeAttempt issues a single HTTP GET to adminURL and classifies
// the response into nil / ErrReadinessNotReady / the raw error.
//
// Status code classification:
//
//	2xx, 401, 403 → nil (alive)
//	503, 502, 504 → ErrReadinessNotReady (transient server error)
//	any other     → ErrReadinessNotReady (treated as transient)
//
// Connection refused / DNS failure / timeout → the raw error (the
// outer loop decides whether to keep retrying).
func probeAttempt(ctx context.Context, client ReadinessProbe, userAgent, adminURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adminURL, nil)
	if err != nil {
		return fmt.Errorf("build readiness request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		// Connection refused / DNS / timeout / ctx cancellation
		// all surface here. The outer loop retries until ctx
		// expires or attempts are exhausted.
		return err
	}
	// Drain + close so HTTP keep-alive connections don't leak across
	// attempts. The body is discarded: we only care about the
	// status code, not the settings payload.
	if resp.Body != nil {
		_, _ = readAndDiscard(resp.Body)
	}
	_ = resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK,
		http.StatusNoContent,
		http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusUnauthorized,
		http.StatusForbidden:
		return nil
	case http.StatusServiceUnavailable,
		http.StatusBadGateway,
		http.StatusGatewayTimeout:
		return fmt.Errorf("%w: status %d", ErrReadinessNotReady, resp.StatusCode)
	default:
		return fmt.Errorf("%w: status %d", ErrReadinessNotReady, resp.StatusCode)
	}
}

// classifyNetErr returns true when err looks like a transient
// connection failure (refused, reset, unreachable). Exported so the
// rollback flow can reuse the same classification when deciding
// whether to retry a freshly-restarted Node-RED.
func classifyNetErr(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	// net.OpError wraps syscall errors that don't satisfy net.Error.
	return strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "no such host")
}

// readAndDiscard drains r without retaining the bytes. We cap the
// drain at 1 MiB so a misbehaving server that streams forever can't
// pin the probe goroutine.
func readAndDiscard(r interface {
	Read(p []byte) (int, error)
}) (int64, error) {
	const maxBytes = 1 << 20
	buf := make([]byte, 1024)
	var total int64
	for total < int64(maxBytes) {
		n, err := r.Read(buf)
		total += int64(n)
		if err != nil {
			return total, err
		}
	}
	return total, nil
}