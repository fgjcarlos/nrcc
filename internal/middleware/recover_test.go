package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// panicHandler is a handler that always panics with the given value.
func panicHandler(panicVal interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		panic(panicVal)
	}
}

// okHandler is a healthy handler that responds 200.
func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// TestRecoverer_PanickingHandler_Returns500 checks that a panicking handler
// yields HTTP 500 with a JSON error body.
func TestRecoverer_PanickingHandler_Returns500(t *testing.T) {
	handler := Recoverer(panicHandler("test panic"))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	// Body must be valid JSON with an "error" key inside "data".
	var resp struct {
		Data struct {
			Error string `json:"error"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("body must be valid JSON: %v\nbody: %s", err, w.Body.String())
	}
	if resp.Data.Error != "internal server error" {
		t.Errorf("error = %q, want %q", resp.Data.Error, "internal server error")
	}
}

// TestRecoverer_ServerSurvivesPanic checks that after a panicking request the
// server is still alive and a subsequent healthy request succeeds.
func TestRecoverer_ServerSurvivesPanic(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/panic", Recoverer(panicHandler("boom")))
	mux.Handle("/ok", Recoverer(http.HandlerFunc(okHandler)))

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// First request — should panic and return 500.
	resp, err := http.Get(srv.URL + "/panic")
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("panic request: status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}

	// Second request to a healthy route — server must still be alive.
	resp2, err := http.Get(srv.URL + "/ok")
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("healthy request after panic: status = %d, want %d", resp2.StatusCode, http.StatusOK)
	}
}

// TestRecoverer_NoStackLeakToClient ensures the response body contains no
// stack trace material: no "goroutine", no file-paths, no ".go:" markers.
func TestRecoverer_NoStackLeakToClient(t *testing.T) {
	handler := Recoverer(panicHandler("secret panic value"))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()

	forbidden := []string{"goroutine", ".go:", "runtime/debug", "secret panic value"}
	for _, sub := range forbidden {
		if strings.Contains(body, sub) {
			t.Errorf("response body leaks %q to client\nbody: %s", sub, body)
		}
	}
}

// TestRecoverer_PanicInLaterMiddlewareCaught tests that a panic raised inside
// a middleware that runs after Recoverer is still caught (ordering guarantee).
func TestRecoverer_PanicInLaterMiddlewareCaught(t *testing.T) {
	// A middleware that panics before calling next.
	panicMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("middleware panic")
		})
	}

	// Stack: Recoverer → panicMiddleware → okHandler
	handler := Recoverer(panicMiddleware(http.HandlerFunc(okHandler)))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("panic in later middleware: status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
