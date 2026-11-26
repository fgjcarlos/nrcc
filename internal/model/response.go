package model

import (
	"encoding/json"
	"net/http"
	"reflect"
	"time"
)

// ApiResponse is the standard response envelope for all API endpoints
type ApiResponse[T any] struct {
	Success   bool      `json:"success"`
	Data      T         `json:"data,omitempty"`
	Error     *ApiError `json:"error,omitempty"`
	Timestamp string    `json:"timestamp"`
}

// ApiError represents an error in the API response
type ApiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ApiErrorResponse is the error response envelope (used for testing/unmarshaling error responses)
type ApiErrorResponse struct {
	Success   bool     `json:"success"`
	Error     *ApiError `json:"error,omitempty"`
	Timestamp string   `json:"timestamp"`
	Code      string   `json:"code,omitempty"` // For backward compatibility
}

// RespondJSON writes a success response as JSON
func RespondJSON[T any](w http.ResponseWriter, status int, data T) {
	// ApiResponse's omitempty would drop an empty list, and clients treat a
	// missing data field as a failed request: always emit data, nil slice as [].
	if v := reflect.ValueOf(&data).Elem(); v.Kind() == reflect.Slice && v.IsNil() {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	}
	resp := struct {
		Success   bool   `json:"success"`
		Data      T      `json:"data"`
		Timestamp string `json:"timestamp"`
	}{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// RespondError writes an error response as JSON
func RespondError(w http.ResponseWriter, status int, code, message string) {
	resp := ApiResponse[any]{
		Success:   false,
		Error:     &ApiError{Code: code, Message: message},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// NowISO8601 returns current time in ISO8601 format
func NowISO8601() string {
	return time.Now().UTC().Format(time.RFC3339)
}
