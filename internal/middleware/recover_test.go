package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nrcc/internal/model"
)

func TestRecovererReturnsJSONError(t *testing.T) {
	t.Parallel()

	handler := RequestID(Recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}

	var resp model.APIResponse[any]
	decodeMiddlewareResponse(t, rec.Body.Bytes(), &resp)
	if resp.Error == nil || resp.Error.Code != "INTERNAL_SERVER_ERROR" {
		t.Fatalf("error payload = %+v", resp.Error)
	}
	if resp.RequestID == "" || resp.Error.RequestID == "" {
		t.Fatalf("request ids missing in recovery response: %+v", resp)
	}
}

func decodeMiddlewareResponse[T any](t *testing.T, payload []byte, target *T) {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode response error = %v body=%s", err, string(payload))
	}
}
