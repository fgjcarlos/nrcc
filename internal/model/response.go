package model

import "time"

type APIResponse[T any] struct {
	Success   bool      `json:"success"`
	Data      T         `json:"data,omitempty"`
	Error     *APIError `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
}

type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	Details   any    `json:"details,omitempty"`
}
