package model

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return body
}

// An empty list must still carry "data": [] — clients treat a missing data
// field as a failed request (the env vars page showed a load error).
func TestRespondJSONEmptyListKeepsData(t *testing.T) {
	cases := map[string][]string{"nil slice": nil, "empty slice": {}}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			RespondJSON(rec, http.StatusOK, data)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			body := decodeEnvelope(t, rec)
			if got := string(body["data"]); got != "[]" {
				t.Fatalf("data = %q, want [] (body %s)", got, rec.Body.String())
			}
			if got := string(body["success"]); got != "true" {
				t.Fatalf("success = %q, want true", got)
			}
		})
	}
}

func TestRespondJSONNonEmptyPayloadUnchanged(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondJSON(rec, http.StatusCreated, []string{"a"})
	body := decodeEnvelope(t, rec)
	if rec.Code != http.StatusCreated || string(body["data"]) != `["a"]` {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if _, ok := body["error"]; ok {
		t.Fatalf("success envelope must not carry error: %s", rec.Body.String())
	}
}

func TestRespondErrorOmitsData(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondError(rec, http.StatusBadRequest, "BAD", "nope")
	body := decodeEnvelope(t, rec)
	if _, ok := body["data"]; ok {
		t.Fatalf("error envelope must not carry data: %s", rec.Body.String())
	}
	if rec.Code != http.StatusBadRequest || string(body["success"]) != "false" {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}
