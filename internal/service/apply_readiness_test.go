package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
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

// refusedAddr returns a loopback address nothing listens on, so a
// connection to it is refused at once.
func refusedAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return addr
}

// TestProbeAdminReady_WaitsForLateListener is the issue #1011
// acceptance case: Node-RED starts accepting connections 15s after the
// restart, and the default deadline still sees it come up.
func TestProbeAdminReady_WaitsForLateListener(t *testing.T) {
	if testing.Short() {
		t.Skip("waits 15s for a late listener")
	}
	addr := refusedAddr(t)
	served := make(chan *http.Server, 1)
	go func() {
		time.Sleep(15 * time.Second)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			served <- nil
			return
		}
		srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}), ReadHeaderTimeout: time.Second}
		served <- srv
		_ = srv.Serve(ln)
	}()
	start := time.Now()
	err := probeAdminReady(context.Background(), "http://"+addr+"/settings")
	elapsed := time.Since(start)
	srv := <-served
	if srv == nil {
		t.Fatalf("could not listen on %s after the delay", addr)
	}
	defer srv.Close()
	if err != nil {
		t.Fatalf("probeAdminReady after %v: %v", elapsed, err)
	}
	if elapsed < 15*time.Second {
		t.Errorf("probe returned after %v, before the listener started", elapsed)
	}
}

// TestProbeAdminReady_FailsAtDeadlineNotBefore asserts a listener that
// never starts fails the probe once the deadline passes, not earlier,
// and keeps the last connection error in the message.
func TestProbeAdminReady_FailsAtDeadlineNotBefore(t *testing.T) {
	const deadline = time.Second
	start := time.Now()
	err := probeAdminReady(context.Background(), "http://"+refusedAddr(t)+"/settings",
		withReadinessDeadline(deadline),
	)
	elapsed := time.Since(start)
	if !errors.Is(err, ErrReadinessNotReady) {
		t.Fatalf("err = %v, want ErrReadinessNotReady", err)
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("err = %v, want the last connection error", err)
	}
	if elapsed < deadline {
		t.Errorf("probe gave up after %v, before the %v deadline", elapsed, deadline)
	}
	if elapsed > deadline+500*time.Millisecond {
		t.Errorf("probe gave up after %v, well past the %v deadline", elapsed, deadline)
	}
}

// TestProbeAdminReady_DeadlineFromEnv asserts NRCC_APPLY_READINESS_TIMEOUT
// sets the deadline when the caller passes none.
func TestProbeAdminReady_DeadlineFromEnv(t *testing.T) {
	t.Setenv(ReadinessTimeoutEnv, "300ms")
	start := time.Now()
	err := probeAdminReady(context.Background(), "http://"+refusedAddr(t)+"/settings")
	elapsed := time.Since(start)
	if !errors.Is(err, ErrReadinessNotReady) {
		t.Fatalf("err = %v, want ErrReadinessNotReady", err)
	}
	if elapsed < 300*time.Millisecond || elapsed > time.Second {
		t.Errorf("probe gave up after %v, want about 300ms", elapsed)
	}
}

func TestReadinessTimeoutFromEnv(t *testing.T) {
	cases := []struct {
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{raw: "", want: 30 * time.Second},
		{raw: "45s", want: 45 * time.Second},
		{raw: " 2m ", want: 2 * time.Minute},
		{raw: "30", wantErr: true},
		{raw: "soon", wantErr: true},
		{raw: "0s", wantErr: true},
		{raw: "-5s", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			t.Setenv(ReadinessTimeoutEnv, tc.raw)
			got, err := ReadinessTimeoutFromEnv()
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), ReadinessTimeoutEnv) {
					t.Fatalf("err = %v, want an error naming %s", err, ReadinessTimeoutEnv)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %v, %v; want %v", got, err, tc.want)
			}
		})
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