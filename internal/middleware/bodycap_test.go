package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/handler"
	"github.com/fgjcarlos/nrcc/internal/middleware"
)

const bodyTooLargeJSON = `{"error":"request body too large"}`

func jsonBody(size int) string {
	return `{"data":"` + strings.Repeat("x", size) + `"}`
}

func decodingHandler(cfg middleware.BodyLimitConfig) http.Handler {
	return middleware.BodyLimitMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Data string `json:"data"`
		}
		if !handler.DecodeJSON(w, r, &payload) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
}

func assertBodyLimitResponse(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := rec.Body.String(); got != bodyTooLargeJSON {
		t.Errorf("body = %q, want %q", got, bodyTooLargeJSON)
	}
}

func TestBodyLimit_Default_RejectsLargeBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(jsonBody(2<<20)))
	rec := httptest.NewRecorder()

	decodingHandler(middleware.DefaultBodyLimitConfig()).ServeHTTP(rec, req)

	assertBodyLimitResponse(t, rec)
}

func TestBodyLimit_Override_AcceptsLargeBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/env/bulk", strings.NewReader(jsonBody(4<<20)))
	rec := httptest.NewRecorder()

	decodingHandler(middleware.DefaultBodyLimitConfig()).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestBodyLimit_Override_RejectsOverriddenSize(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/env/bulk", strings.NewReader(jsonBody(6<<20)))
	rec := httptest.NewRecorder()

	decodingHandler(middleware.DefaultBodyLimitConfig()).ServeHTTP(rec, req)

	assertBodyLimitResponse(t, rec)
}

func TestBodyLimit_DefaultOverrides(t *testing.T) {
	want := map[string]int64{
		http.MethodPost + " /api/env/bulk":        5 << 20,
		http.MethodPut + " /api/env/dotenv":       5 << 20,
		http.MethodPost + " /api/settings/raw":    2 << 20,
		http.MethodPost + " /api/ai/analyze/flow": 5 << 20,
	}
	cfg := middleware.DefaultBodyLimitConfig()
	if cfg.DefaultLimit != 1<<20 {
		t.Errorf("default limit = %d, want %d", cfg.DefaultLimit, 1<<20)
	}
	if len(cfg.Overrides) != len(want) {
		t.Fatalf("override count = %d, want %d", len(cfg.Overrides), len(want))
	}
	for route, limit := range want {
		if got := cfg.Overrides[route]; got != limit {
			t.Errorf("override %q = %d, want %d", route, got, limit)
		}
	}

	cfg.Overrides[http.MethodPost+" /api/env/bulk"] = 1
	fresh := middleware.DefaultBodyLimitConfig()
	if got := fresh.Overrides[http.MethodPost+" /api/env/bulk"]; got != 5<<20 {
		t.Errorf("fresh override map was mutated: got %d", got)
	}
}

func TestBodyLimit_CopiesOverridePolicy(t *testing.T) {
	cfg := middleware.BodyLimitConfig{
		DefaultLimit: 8,
		Overrides:    map[string]int64{http.MethodPost + " /api/test": 16},
	}
	h := decodingHandler(cfg)
	cfg.Overrides[http.MethodPost+" /api/test"] = 1
	req := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader(jsonBody(4)))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

type trackingBody struct {
	*strings.Reader
}

func (b *trackingBody) Close() error { return nil }

func TestBodyLimit_Multipart_Excluded(t *testing.T) {
	body := &trackingBody{Reader: strings.NewReader(strings.Repeat("x", 10<<20))}
	var gotBody io.ReadCloser
	var gotBytes int64
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody = r.Body
		var err error
		gotBytes, err = io.Copy(io.Discard, r.Body)
		if err != nil {
			t.Fatalf("read multipart body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/files/upload", nil)
	req.Body = body
	req.ContentLength = 10 << 20
	req.Header.Set("Content-Type", "  MuLtIpArT/form-data; boundary=test")
	rec := httptest.NewRecorder()

	middleware.BodyLimitMiddleware(middleware.DefaultBodyLimitConfig())(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if gotBody != body {
		t.Error("multipart request body was replaced")
	}
	if gotBytes != 10<<20 {
		t.Errorf("bytes read = %d, want %d", gotBytes, 10<<20)
	}
}

func TestBodyLimit_ContentLengthMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(jsonBody(2<<20)))
	req.ContentLength = -1
	req.TransferEncoding = []string{"chunked"}
	rec := httptest.NewRecorder()

	decodingHandler(middleware.DefaultBodyLimitConfig()).ServeHTTP(rec, req)

	assertBodyLimitResponse(t, rec)
}

func TestBodyLimit_ResponseIsJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(jsonBody(2<<20)))
	rec := httptest.NewRecorder()

	decodingHandler(middleware.DefaultBodyLimitConfig()).ServeHTTP(rec, req)

	assertBodyLimitResponse(t, rec)
}

func TestBodyLimit_OverrideRequiresExactMethodAndPath(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "method mismatch", method: http.MethodPut, path: "/api/env/bulk"},
		{name: "path mismatch", method: http.MethodPost, path: "/api/env/bulk/extra"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(jsonBody(2<<20)))
			rec := httptest.NewRecorder()

			decodingHandler(middleware.DefaultBodyLimitConfig()).ServeHTTP(rec, req)

			assertBodyLimitResponse(t, rec)
		})
	}
}

func TestBodyLimit_MissingOrMisleadingContentTypeStillCapped(t *testing.T) {
	for _, contentType := range []string{"", "text/plain", "multipartish/form-data"} {
		t.Run(contentType, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/config", strings.NewReader(jsonBody(2<<20)))
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()

			decodingHandler(middleware.DefaultBodyLimitConfig()).ServeHTTP(rec, req)

			assertBodyLimitResponse(t, rec)
		})
	}
}
