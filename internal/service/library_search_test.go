package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// TestSearchRespectsContextCancel is the S-REQ-3.1 regression for HIGH-013:
// `LibraryService.Search` used `http.Get(...)` with no client, no timeout, and
// no request context. If the caller cancels before the request begins, the
// function must return immediately.
func TestSearchRespectsContextCancel(t *testing.T) {
	svc := NewLibraryService(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	res, err := svc.Search(ctx, "node-red")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("Search: expected empty results on canceled ctx, got %d entries", len(res))
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("Search did not honor pre-canceled context in <500ms (took %s)", elapsed)
	}
}

// TestSearchTimesOutAgainstStalledRegistry is the S-REQ-3.2 regression for
// HIGH-013: even when the parent context has no deadline, the HTTP client
// timeout must bound the wait. We stand up an httptest server that sleeps
// 60s, point a custom client at it (timeout 200ms), and verify Search returns
// within 1s with an empty result.
func TestSearchTimesOutAgainstStalledRegistry(t *testing.T) {
	stalled := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the client cancels the request, then return immediately.
		// This avoids the 60s cleanup delay that httptest.Server.Close waits
		// for when an in-flight handler is blocked on time.Sleep.
		<-r.Context().Done()
	}))
	defer stalled.Close()

	// Rewrite the registry URL to point at our stalled server by overriding
	// the http.Client transport (RoundTripper URL rewriting).
	svc := NewLibraryService(t.TempDir()).WithHTTPClient(&http.Client{
		Timeout: 200 * time.Millisecond,
		Transport: &rewriteTransport{
			base:     http.DefaultTransport,
			fromHost: "registry.npmjs.org",
			fromPath: "/-/v1/search",
			to:       stalled.URL,
		},
	})

	start := time.Now()
	res, err := svc.Search(context.Background(), "node-red")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("Search: expected empty results on stalled upstream, got %d entries", len(res))
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Search took %s with a 200ms client timeout; client.Timeout is not enforced", elapsed)
	}
}

// rewriteTransport rewrites every request whose path starts with `fromPath`
// to the absolute URL in `to`. Used to redirect the production npm registry
// URL to an httptest server. Matching is on the path + host (not the full
// URL with query string) because Search appends query params at runtime.
type rewriteTransport struct {
	base     http.RoundTripper
	fromHost string
	fromPath string
	to       string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == t.fromHost && req.URL.Path == t.fromPath {
		newReq := req.Clone(req.Context())
		newURL, _ := url.Parse(t.to)
		newURL.RawQuery = req.URL.RawQuery
		newReq.URL = newURL
		newReq.Host = newURL.Host
		req = newReq
	}
	return t.base.RoundTrip(req)
}

// TestSearchSucceedsAgainstFastRegistry is the positive-path regression: when
// the registry responds quickly with valid JSON, Search returns the parsed
// entries (non-empty).
func TestSearchSucceedsAgainstFastRegistry(t *testing.T) {
	body := `{"objects":[{"package":{"name":"node-red-dashboard","version":"3.6.0","description":"dashboard","date":"2024-01-01T00:00:00.000Z","keywords":["dashboard"],"links":{"npm":"https://www.npmjs.com/package/node-red-dashboard","homepage":"https://github.com/...","repository":"https://github.com/..."}},"downloads":{"weekly":12345}}]}`
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	defer fast.Close()

	svc := NewLibraryService(t.TempDir()).WithHTTPClient(&http.Client{
		Timeout: 5 * time.Second,
		Transport: &rewriteTransport{
			base:     http.DefaultTransport,
			fromHost: "registry.npmjs.org",
			fromPath: "/-/v1/search",
			to:       fast.URL,
		},
	})

	res, err := svc.Search(context.Background(), "node-red-dashboard")
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("Search: expected 1 result, got %d", len(res))
	}
	if res[0].Name != "node-red-dashboard" {
		t.Fatalf("Search: expected name=node-red-dashboard, got %q", res[0].Name)
	}
	if res[0].Downloads != 12345 {
		t.Fatalf("Search: expected Downloads=12345, got %d", res[0].Downloads)
	}
}

// TestLibrarySearchDefaultsToBoundedHTTPClient verifies the default HTTP client
// constructed by NewLibraryService has a finite Timeout (the production
// `30s` constant). Without this guard, a developer could regress to
// `http.DefaultClient` (no timeout) and reintroduce HIGH-013.
func TestLibrarySearchDefaultsToBoundedHTTPClient(t *testing.T) {
	svc := NewLibraryService(t.TempDir())
	if svc.httpClient == nil {
		t.Fatal("NewLibraryService must initialize a non-nil httpClient")
	}
	if svc.httpClient.Timeout <= 0 {
		t.Fatalf("NewLibraryService must set httpClient.Timeout > 0 (got %s); reintroduces HIGH-013", svc.httpClient.Timeout)
	}
}
