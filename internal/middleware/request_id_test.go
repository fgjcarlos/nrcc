package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestRequestID_IgnoresInboundValues(t *testing.T) {
	for _, inbound := range []string{"", uuid.NewString(), "not-a-uuid"} {
		t.Run(inbound, func(t *testing.T) {
			var contextID string
			h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contextID = RequestIDFromContext(r.Context())
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			req.Header.Set("X-Request-Id", inbound)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			headerID := rec.Header().Get("X-Request-Id")
			if contextID == "" || headerID != contextID {
				t.Fatalf("context ID %q and header ID %q must be equal and non-empty", contextID, headerID)
			}
			parsed, err := uuid.Parse(contextID)
			if err != nil || parsed.Version() != 4 {
				t.Fatalf("request ID %q is not UUID v4: %v", contextID, err)
			}
			if inbound != "" && contextID == inbound {
				t.Fatalf("server reused inbound request ID %q", inbound)
			}
		})
	}
}

func TestEnsureRequestMetadata_IsIdempotentAndNonEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	firstReq, first := ensureRequestMetadata(req)
	secondReq, second := ensureRequestMetadata(firstReq)
	if first.requestID == "" {
		t.Fatal("request ID must not be empty")
	}
	if first != second || firstReq != secondReq {
		t.Fatal("ensureRequestMetadata must reuse request-owned metadata")
	}
}

func TestRequestID_ConcurrentRequestsAreIsolated(t *testing.T) {
	const count = 48
	ids := make(chan string, count)
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids <- RequestIDFromContext(r.Context())
	}))
	var wg sync.WaitGroup
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			if got := rec.Header().Get("X-Request-Id"); got == "" {
				t.Error("response request ID must not be empty")
			}
		}()
	}
	wg.Wait()
	close(ids)
	seen := make(map[string]bool, count)
	for id := range ids {
		if seen[id] {
			t.Errorf("duplicate request ID %q", id)
		}
		seen[id] = true
	}
	if len(seen) != count {
		t.Fatalf("got %d unique IDs, want %d", len(seen), count)
	}
}
