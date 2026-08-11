package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON_RejectsMalformedJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-json"))
	rec := httptest.NewRecorder()

	if DecodeJSON(rec, req, &struct{}{}) {
		t.Fatal("DecodeJSON returned true for malformed JSON")
	}
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDecodeJSON_TranslatesMaxBytesError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"data":"too large"}`))
	rec := httptest.NewRecorder()
	req.Body = http.MaxBytesReader(rec, req.Body, 8)

	if DecodeJSON(rec, req, &struct{}{}) {
		t.Fatal("DecodeJSON returned true for an oversized body")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := rec.Body.String(); got != `{"error":"request body too large"}` {
		t.Errorf("body = %q, want exact overflow contract", got)
	}
}

func TestDecodeJSONOptional_AllowsEmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	if !DecodeJSONOptional(rec, req, &struct{}{}) {
		t.Fatalf("DecodeJSONOptional rejected an empty body with status %d", rec.Code)
	}
}

func TestDecodeJSONBestEffort_PreservesMalformedBodyBehavior(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-json"))
	rec := httptest.NewRecorder()

	if !DecodeJSONBestEffort(rec, req, &struct{}{}) {
		t.Fatalf("DecodeJSONBestEffort rejected malformed JSON with status %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("best-effort decode wrote response %q", rec.Body.String())
	}
}

func TestDecodeJSONBestEffort_StillRejectsOversizedBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"data":"too large"}`))
	rec := httptest.NewRecorder()
	req.Body = http.MaxBytesReader(rec, req.Body, 8)

	if DecodeJSONBestEffort(rec, req, &struct{}{}) {
		t.Fatal("DecodeJSONBestEffort returned true for an oversized body")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}
