package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/fgjcarlos/nrcc/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type failingSSEWriter struct{ header http.Header }

func (w *failingSSEWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}
func (*failingSSEWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }
func (*failingSSEWriter) WriteHeader(int)           {}
func (*failingSSEWriter) Flush()                    {}
func (*failingSSEWriter) FlushError() error         { return errors.New("closed") }

func TestWriteSSEStopsOnDisconnectedClient(t *testing.T) {
	if writeSSE(&failingSSEWriter{}, httptest.NewRecorder(), "data: x\\n\\n") {
		t.Fatal("writeSSE should stop after failed write")
	}
}

func TestRuntimeLogsRejectsInvalidLimitAndGeneration(t *testing.T) {
	pm := service.NewProcessManager("node-red", t.TempDir())
	h := NewRuntimeLogsHandler(pm)
	generation := pm.RuntimeLogs("", 1, nil, "", time.Time{}, time.Time{}).Generation
	for _, target := range []string{"/api/logs?limit=1001", "/api/logs?generation=stale", "/api/logs?cursor=" + generation + ":bogus"} {
		rec := httptest.NewRecorder()
		h.History(rec, httptest.NewRequest(http.MethodGet, target, nil))
		want := 400
		if strings.Contains(target, "generation=stale") {
			want = 409
		}
		if strings.Contains(target, "cursor=") {
			want = 400
		}
		if rec.Code != want {
			t.Errorf("%s status=%d want=%d", target, rec.Code, want)
		}
	}
}

func TestRuntimeLogsHistoryAndStreamSnapshot(t *testing.T) {
	pm := service.NewProcessManager("node-red", t.TempDir())
	h := NewRuntimeLogsHandler(pm)
	// Buffer is exercised through the public process API's forwarding-independent capture seam.
	pm.RuntimeLogBuffer().AddForTest("12 Jan 12:34:56 - [error] failure")
	rec := httptest.NewRecorder()
	h.History(rec, httptest.NewRequest(http.MethodGet, "/api/logs?limit=1", nil))
	if rec.Code != 200 {
		t.Fatalf("history status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("raw history cache policy=%q", rec.Header().Get("Cache-Control"))
	}
	var result struct {
		Data struct {
			Entries       []service.RuntimeLogEntry `json:"entries"`
			HighWatermark string                    `json:"highWatermark"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data.Entries) != 1 || result.Data.Entries[0].Source != "stdout" {
		t.Fatalf("unexpected response: %#v", result)
	}
	beforeFirst := httptest.NewRecorder()
	h.History(beforeFirst, httptest.NewRequest(http.MethodGet, "/api/logs?cursor="+strings.Replace(result.Data.HighWatermark, ":1", ":0", 1), nil))
	if beforeFirst.Code != http.StatusOK || !strings.Contains(beforeFirst.Body.String(), `"entries":[]`) {
		t.Fatalf("paging before sequence zero returned %d: %s", beforeFirst.Code, beforeFirst.Body.String())
	}
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/logs/stream?since="+result.Data.HighWatermark, nil).WithContext(ctx)
	stream := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { h.Stream(stream, req); close(done) }()
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-done
	if !strings.Contains(stream.Body.String(), "event: connected") || strings.Contains(stream.Body.String(), `"startedAt"`) {
		t.Fatalf("invalid initial event before startup: %s", stream.Body.String())
	}
}
