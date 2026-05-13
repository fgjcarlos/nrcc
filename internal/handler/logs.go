package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
)

// LogHandler handles Node-RED log endpoints
type LogHandler struct {
	logBuffer *service.LogBuffer
}

// NewLogHandler creates a new LogHandler
func NewLogHandler(logBuffer *service.LogBuffer) *LogHandler {
	return &LogHandler{
		logBuffer: logBuffer,
	}
}

// GetLogs returns recent log entries (GET /api/runtime/logs)
func (h *LogHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	tail := 100
	if tailStr := r.URL.Query().Get("tail"); tailStr != "" {
		if t, err := strconv.Atoi(tailStr); err == nil && t > 0 && t <= 1000 {
			tail = t
		}
	}

	entries := h.logBuffer.Recent(tail)
	model.RespondJSON(w, http.StatusOK, entries)
}

// StreamLogs streams log entries via SSE (GET /api/runtime/logs/stream)
func (h *LogHandler) StreamLogs(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Parse replay parameter (number of recent logs to send first)
	replay := 100
	if replayStr := r.URL.Query().Get("replay"); replayStr != "" {
		if r, err := strconv.Atoi(replayStr); err == nil && r > 0 && r <= 1000 {
			replay = r
		}
	}

	// Send buffered count event
	fmt.Fprintf(w, "event: connected\n")
	data := map[string]int{"bufferedCount": h.logBuffer.Count()}
	if buf, err := json.Marshal(data); err == nil {
		fmt.Fprintf(w, "data: %s\n\n", string(buf))
	}

	// Send recent entries
	recentEntries := h.logBuffer.Recent(replay)
	for _, entry := range recentEntries {
		if buf, err := json.Marshal(entry); err == nil {
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", string(buf))
		}
	}

	// Flush the response
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Subscribe to new entries
	ch, unsub := h.logBuffer.Subscribe()
	defer unsub()

	// Send new entries as they arrive
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case entry, ok := <-ch:
			if !ok {
				return // Channel closed
			}
			if buf, err := json.Marshal(entry); err == nil {
				fmt.Fprintf(w, "event: log\ndata: %s\n\n", string(buf))
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}

		case <-ticker.C:
			// Send keepalive ping
			fmt.Fprintf(w, ": ping\n\n")
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}

// DeleteLogs clears the log buffer (DELETE /api/runtime/logs)
func (h *LogHandler) DeleteLogs(w http.ResponseWriter, r *http.Request) {
	h.logBuffer.Clear()
	w.WriteHeader(http.StatusNoContent)
}
