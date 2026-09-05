package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fakeReadinessProbe is the testing double for ReadinessProbe. Each
// canned response is queued; calls pop from the front and recurse to
// the next one. When the queue is empty the probe returns
// ErrReadinessNotReady so tests can drive the retry loop.
type fakeReadinessProbe struct {
	responses []probeResponse
	calls     int32
}

type probeResponse struct {
	status int
	err    error
}

func (f *fakeReadinessProbe) Do(req *http.Request) (*http.Response, error) {
	idx := int(atomic.AddInt32(&f.calls, 1)) - 1
	if idx >= len(f.responses) {
		return nil, fmt.Errorf("no canned response for call %d", idx+1)
	}
	resp := f.responses[idx]
	if resp.err != nil {
		return nil, resp.err
	}
	return &http.Response{
		StatusCode: resp.status,
		Body:       http.NoBody,
	}, nil
}

// TestProbeAdminReady_HappyPath asserts a single 200 response is
// sufficient to declare ready (no retries, no backoff).
func TestProbeAdminReady_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/settings" {
			t.Errorf("probe path = %q, want /settings", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	probe := &fakeReadinessProbe{responses: []probeResponse{{status: http.StatusOK}}}
	if err := probeAdminReady(context.Background(), srv.URL+"/settings", withReadinessProbe(probe)); err != nil {
		t.Fatalf("probeAdminReady: %v", err)
	}
	if got := atomic.LoadInt32(&probe.calls); got != 1 {
		t.Errorf("probe calls = %d, want 1", got)
	}
}

// TestProbeAdminReady_StatusMatrix locks in the alive / notYetReady
// classification documented in apply_readiness.go.
func TestProbeAdminReady_StatusMatrix(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		wantAlive   bool
	}{
		{name: "200IsAlive", status: http.StatusOK, wantAlive: true},
		{name: "204IsAlive", status: http.StatusNoContent, wantAlive: true},
		{name: "301IsAlive", status: http.StatusMovedPermanently, wantAlive: true},
		{name: "302IsAlive", status: http.StatusFound, wantAlive: true},
		{name: "401IsAlive", status: http.StatusUnauthorized, wantAlive: true},
		{name: "403IsAlive", status: http.StatusForbidden, wantAlive: true},
		{name: "503IsNotAlive", status: http.StatusServiceUnavailable, wantAlive: false},
		{name: "502IsNotAlive", status: http.StatusBadGateway, wantAlive: false},
		{name: "500IsNotAlive", status: http.StatusInternalServerError, wantAlive: false},
		{name: "404IsNotAlive", status: http.StatusNotFound, wantAlive: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()
			probe := &fakeReadinessProbe{responses: []probeResponse{{status: tc.status}}}
			err := probeAdminReady(context.Background(), srv.URL, withReadinessProbe(probe))
			if tc.wantAlive {
				if err != nil {
					t.Errorf("alive status %d returned err = %v, want nil", tc.status, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("not-alive status %d returned nil err, want ErrReadinessNotReady", tc.status)
			}
			if !errors.Is(err, ErrReadinessNotReady) {
				t.Errorf("err = %v, want ErrReadinessNotReady", err)
			}
		})
	}
}

// TestProbeAdminReady_RetryUntilAlive asserts the probe loops through
// transient connection-refused responses until a 200 arrives. The
// total probe count is what the test asserts; the backoff window is
// kept short (200ms) so the test runs in well under a second.
func TestProbeAdminReady_RetryUntilAlive(t *testing.T) {
	probe := &fakeReadinessProbe{responses: []probeResponse{
		{err: errors.New("connection refused")},
		{err: errors.New("connection refused")},
		{err: errors.New("connection refused")},
		{status: http.StatusOK},
	}}
	if err := probeAdminReady(context.Background(), "http://127.0.0.1:1/settings",
		withReadinessProbe(probe),
		withReadinessBackoff(10*time.Millisecond),
		withReadinessTimeout(50*time.Millisecond),
	); err != nil {
		t.Fatalf("probeAdminReady: %v", err)
	}
	if got := atomic.LoadInt32(&probe.calls); got != 4 {
		t.Errorf("probe calls = %d, want 4 (3 retries + 1 success)", got)
	}
}

// TestProbeAdminReady_ExhaustionReturnsErrReadinessNotReady asserts
// the probe exhausts its attempts and wraps the last underlying error.
func TestProbeAdminReady_ExhaustionReturnsErrReadinessNotReady(t *testing.T) {
	probe := &fakeReadinessProbe{responses: []probeResponse{
		{err: errors.New("connection refused")},
		{err: errors.New("connection refused")},
		{err: errors.New("connection refused")},
	}}
	err := probeAdminReady(context.Background(), "http://127.0.0.1:1/settings",
		withReadinessProbe(probe),
		withReadinessAttempts(3),
		withReadinessBackoff(10*time.Millisecond),
		withReadinessTimeout(50*time.Millisecond),
	)
	if !errors.Is(err, ErrReadinessNotReady) {
		t.Errorf("err = %v, want ErrReadinessNotReady", err)
	}
	if got := atomic.LoadInt32(&probe.calls); got != 3 {
		t.Errorf("probe calls = %d, want 3 (attempts exhausted)", got)
	}
}

// TestProbeAdminReady_ContextCancellationStopsProbe asserts a cancelled
// context short-circuits the retry loop before the next attempt and
// surfaces ctx.Err() to the caller.
func TestProbeAdminReady_ContextCancellationStopsProbe(t *testing.T) {
	probe := &fakeReadinessProbe{responses: []probeResponse{
		{err: errors.New("connection refused")},
		{err: errors.New("connection refused")},
		{err: errors.New("connection refused")},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after the first probe call returns. We do this with a
	// goroutine that races against the probe; the test's generous
	// deadline (5s) lets us wait for the cancellation to land.
	go func() {
		for atomic.LoadInt32(&probe.calls) < 1 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	err := probeAdminReady(ctx, "http://127.0.0.1:1/settings",
		withReadinessProbe(probe),
		withReadinessBackoff(10*time.Millisecond),
		withReadinessTimeout(50*time.Millisecond),
	)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

// TestProbeAdminReady_EmptyURLRejected asserts the boundary check
// fires before any HTTP work, protecting the orchestrator from an
// invalid Ready call.
func TestProbeAdminReady_EmptyURLRejected(t *testing.T) {
	if err := probeAdminReady(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty URL, got nil")
	}
}

// TestClassifyNetErr_Cases locks in the connection-refused / reset /
// DNS detection used by the rollback flow when deciding whether to
// retry a freshly-restarted Node-RED.
func TestClassifyNetErr_Cases(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nilIsNotNetErr", err: nil, want: false},
		{name: "plainErrorIsNotNetErr", err: errors.New("something"), want: false},
		{name: "refusedIsNetErr", err: errors.New("dial tcp: connection refused"), want: true},
		{name: "resetIsNetErr", err: errors.New("read: connection reset by peer"), want: true},
		{name: "noSuchHostIsNetErr", err: errors.New("lookup foo: no such host"), want: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyNetErr(tc.err); got != tc.want {
				t.Errorf("classifyNetErr(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}