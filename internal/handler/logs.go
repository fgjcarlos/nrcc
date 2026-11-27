package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

type RuntimeLogsHandler struct{ pm *service.ProcessManager }

func NewRuntimeLogsHandler(pm *service.ProcessManager) *RuntimeLogsHandler {
	return &RuntimeLogsHandler{pm: pm}
}
func (h *RuntimeLogsHandler) SetProcessManager(pm *service.ProcessManager) { h.pm = pm }
func (h *RuntimeLogsHandler) History(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h.pm == nil {
		model.RespondError(w, http.StatusServiceUnavailable, "LOGS_UNAVAILABLE", "Runtime logs unavailable")
		return
	}
	q := r.URL.Query()
	limit := 100
	if v := q.Get("limit"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 || n > 1000 {
			model.RespondError(w, http.StatusBadRequest, "INVALID_QUERY", "limit must be between 1 and 1000")
			return
		}
		limit = n
	}
	var from, to time.Time
	var err error
	if q.Get("from") != "" {
		from, err = time.Parse(time.RFC3339, q.Get("from"))
		if err != nil {
			model.RespondError(w, http.StatusBadRequest, "INVALID_QUERY", "from must be RFC3339")
			return
		}
	}
	if q.Get("to") != "" {
		to, err = time.Parse(time.RFC3339, q.Get("to"))
		if err != nil {
			model.RespondError(w, http.StatusBadRequest, "INVALID_QUERY", "to must be RFC3339")
			return
		}
	}
	cursor := q.Get("cursor")
	sinceRestart := false
	if raw := q.Get("sinceRestart"); raw != "" {
		parsed, e := strconv.ParseBool(raw)
		if e != nil {
			model.RespondError(w, http.StatusBadRequest, "INVALID_QUERY", "sinceRestart must be true or false")
			return
		}
		sinceRestart = parsed
	}
	if cursor != "" && !strings.HasPrefix(cursor, h.pm.RuntimeLogs("", 1, nil, "", time.Time{}, time.Time{}).Generation+":") {
		model.RespondError(w, http.StatusConflict, "LOG_GENERATION_CHANGED", "Runtime log generation changed; restart pagination")
		return
	}
	if cursor != "" && !h.pm.RuntimeLogBuffer().ValidCursor(cursor) {
		model.RespondError(w, http.StatusBadRequest, "INVALID_QUERY", "cursor is malformed or beyond the current high watermark")
		return
	}
	page := h.pm.RuntimeLogsSince(cursor, limit, q["level"], q.Get("q"), from, to, sinceRestart)
	if requested := q.Get("generation"); requested != "" && requested != page.Generation {
		model.RespondError(w, http.StatusConflict, "LOG_GENERATION_CHANGED", "Runtime log generation changed; restart pagination")
		return
	}
	model.RespondJSON(w, http.StatusOK, page)
}
func (h *RuntimeLogsHandler) Stream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if h.pm == nil {
		http.Error(w, "Runtime logs unavailable", http.StatusServiceUnavailable)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	cursor := r.Header.Get("Last-Event-ID")
	if cursor == "" {
		cursor = r.URL.Query().Get("since")
	}
	page := h.pm.RuntimeLogs("", 1, nil, "", time.Time{}, time.Time{})
	if cursor != "" && !strings.HasPrefix(cursor, page.Generation+":") {
		http.Error(w, "Runtime log generation changed; reconnect from a fresh history snapshot", http.StatusConflict)
		return
	}
	if cursor != "" && !h.pm.RuntimeLogBuffer().ValidCursor(cursor) {
		http.Error(w, "Malformed or future runtime log cursor", http.StatusBadRequest)
		return
	}
	if cursor == "" {
		cursor = page.HighWatermark
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	if !writeSSE(w, flusher, logSnapshotEvent("connected", page)) {
		return
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	ping := time.NewTicker(15 * time.Second)
	defer ping.Stop()
	authRefresh := time.NewTimer(5 * time.Minute)
	defer authRefresh.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-authRefresh.C:
			return
		case <-ticker.C:
			current := h.pm.RuntimeLogs("", 1, nil, "", time.Time{}, time.Time{})
			if !current.StartedAt.Equal(page.StartedAt) {
				_ = writeSSE(w, flusher, logSnapshotEvent("restart", current))
				return
			}
			for _, entry := range h.pm.RuntimeLogsAfter(cursor) {
				if !writeSSE(w, flusher, "id: "+entry.ID+"\nevent: log\ndata: "+mustJSON(entry)+"\n\n") {
					return
				}
				cursor = entry.ID
			}
		case <-ping.C:
			if !writeSSE(w, flusher, ": ping\n\n") {
				return
			}
		}
	}
}
func logSnapshotEvent(event string, page service.RuntimeLogPage) string {
	meta := struct {
		Generation    string    `json:"generation"`
		HighWatermark string    `json:"highWatermark"`
		StartedAt     time.Time `json:"startedAt,omitzero"`
	}{page.Generation, page.HighWatermark, page.StartedAt}
	return "event: " + event + "\ndata: " + mustJSON(meta) + "\n\n"
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event string) bool {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(20 * time.Second))
	// #nosec G705 -- the stream has a text/event-stream content type; all event payloads are JSON-encoded.
	if _, err := w.Write([]byte(event)); err != nil {
		return false
	}
	if err := http.NewResponseController(w).Flush(); err != nil {
		return false
	}
	flusher.Flush()
	return true
}
func mustJSON(v any) string { b, _ := json.Marshal(v); return strings.TrimSpace(string(b)) }
