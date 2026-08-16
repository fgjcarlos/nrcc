package middleware

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type fullResponseWriter struct{ *httptest.ResponseRecorder }

func (w *fullResponseWriter) Flush() {}
func (w *fullResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errors.New("test hijack")
}
func (w *fullResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(w.ResponseRecorder, r)
}
func (w *fullResponseWriter) Push(string, *http.PushOptions) error { return http.ErrNotSupported }

type recordHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *recordHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *recordHandler) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h *recordHandler) WithGroup(string) slog.Handler            { return h }
func (h *recordHandler) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, record.Clone())
	return nil
}

func (h *recordHandler) only(t *testing.T) (string, map[string]any) {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.records) != 1 {
		t.Fatalf("got %d log records, want exactly 1", len(h.records))
	}
	attrs := make(map[string]any)
	h.records[0].Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value.Any()
		return true
	})
	return h.records[0].Message, attrs
}

func TestLogger_EmitsSafeCompletionRecord(t *testing.T) {
	handler := &recordHandler{}
	logger := slog.New(handler)
	h := Logger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) == "" {
			t.Error("logger-only composition must create request metadata")
		}
		w.WriteHeader(http.StatusTeapot)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/backups?password=secret", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	message, attrs := handler.only(t)
	if message != "http_request_completed" || attrs["method"] != http.MethodGet || attrs["path"] != "/api/backups" {
		t.Fatalf("unexpected completion record: message=%q attrs=%v", message, attrs)
	}
	if attrs["status"] != int64(http.StatusTeapot) || attrs["request_id"] != rec.Header().Get("X-Request-Id") {
		t.Fatalf("status/request ID mismatch: attrs=%v header=%q", attrs, rec.Header().Get("X-Request-Id"))
	}
	if duration, ok := attrs["duration_ms"].(int64); !ok || duration < 0 {
		t.Fatalf("duration_ms must be a non-negative integer, got %#v", attrs["duration_ms"])
	}
	for _, value := range append([]any{message}, attrs["path"]) {
		if value == "secret" || value == "password=secret" {
			t.Fatalf("completion record leaked query secret: %v", value)
		}
	}
}

func TestLogger_StatusTransitionsAndCapabilities(t *testing.T) {
	h := Logger(slog.New(slog.NewTextHandler(io.Discard, nil)))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped, ok := w.(interface{ Status() int })
		if !ok || wrapped.Status() != 0 {
			t.Fatalf("fresh writer status = %v, want 0", wrapped)
		}
		if _, ok := w.(interface{ Unwrap() http.ResponseWriter }); !ok {
			t.Fatal("wrapped writer must expose Unwrap")
		}
		w.WriteHeader(http.StatusEarlyHints)
		if wrapped.Status() != 0 {
			t.Fatalf("informational status = %d, want 0", wrapped.Status())
		}
		_, _ = w.Write([]byte("ok"))
		w.WriteHeader(http.StatusCreated)
		if wrapped.Status() != http.StatusOK {
			t.Fatalf("first final status = %d, want 200", wrapped.Status())
		}
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if _, ok := any(rec).(http.Hijacker); ok {
		t.Fatal("minimal underlying writer must not gain Hijacker")
	}
}

func TestLogger_ProtocolCapabilityMatrix(t *testing.T) {
	for _, test := range []struct {
		name                            string
		proto                           int
		flusher, hijacker, reader, push bool
	}{
		{name: "HTTP1", proto: 1, flusher: true, hijacker: true, reader: true},
		{name: "HTTP2", proto: 2, flusher: true, push: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := Logger(slog.New(slog.NewTextHandler(io.Discard, nil)))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, flusher := w.(http.Flusher)
				_, hijacker := w.(http.Hijacker)
				_, reader := w.(io.ReaderFrom)
				_, push := w.(http.Pusher)
				if flusher != test.flusher || hijacker != test.hijacker || reader != test.reader || push != test.push {
					t.Fatalf("capabilities flusher=%v hijacker=%v reader=%v push=%v", flusher, hijacker, reader, push)
				}
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.ProtoMajor = test.proto
			h.ServeHTTP(&fullResponseWriter{httptest.NewRecorder()}, req)
		})
	}
}

func TestLogger_SwitchingProtocolsIsFinal(t *testing.T) {
	handler := &recordHandler{}
	h := Logger(slog.New(handler))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusSwitchingProtocols)
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/upgrade", nil))
	_, attrs := handler.only(t)
	if attrs["status"] != int64(http.StatusSwitchingProtocols) {
		t.Fatalf("status = %#v, want 101", attrs["status"])
	}
}

func TestLogger_Logs500AndRepanics(t *testing.T) {
	handler := &recordHandler{}
	h := Logger(slog.New(handler))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	defer func() {
		if recover() == nil {
			t.Fatal("logger must re-panic")
		}
		_, attrs := handler.only(t)
		if attrs["status"] != int64(http.StatusInternalServerError) {
			t.Fatalf("panic status = %#v, want 500", attrs["status"])
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/panic", nil))
}
