package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/model"
)

const bodyTooLargeResponse = `{"error":"request body too large"}`

// DecodeJSON decodes a required JSON request body and writes the appropriate client error.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	err := json.NewDecoder(r.Body).Decode(dst)
	if err == nil {
		return true
	}
	if respondBodyLimitError(w, err) {
		return false
	}
	model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	return false
}

// DecodeJSONOptional decodes JSON when present and accepts an empty request body.
func DecodeJSONOptional(w http.ResponseWriter, r *http.Request, dst any) bool {
	err := json.NewDecoder(r.Body).Decode(dst)
	if err == nil || errors.Is(err, io.EOF) {
		return true
	}
	if respondBodyLimitError(w, err) {
		return false
	}
	model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	return false
}

// DecodeJSONBestEffort ignores malformed JSON but still rejects oversized bodies.
func DecodeJSONBestEffort(w http.ResponseWriter, r *http.Request, dst any) bool {
	err := json.NewDecoder(r.Body).Decode(dst)
	return err == nil || !respondBodyLimitError(w, err)
}

func respondBodyLimitError(w http.ResponseWriter, err error) bool {
	var maxBytesErr *http.MaxBytesError
	if !errors.As(err, &maxBytesErr) {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusRequestEntityTooLarge)
	_, _ = w.Write([]byte(bodyTooLargeResponse))
	return true
}
