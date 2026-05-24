package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
)

func TestGetSystemHistory_Returns200WithEmptySlice(t *testing.T) {
	buf := service.NewMetricsBuffer(120)
	h := NewSystemHandler()
	h.SetMetricsBuffer(buf)

	req := httptest.NewRequest(http.MethodGet, "/api/system/history", nil)
	w := httptest.NewRecorder()

	h.GetSystemHistory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp model.ApiResponse[[]model.MetricsSnapshot]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response must be valid JSON: %v", err)
	}

	if !resp.Success {
		t.Error("response success must be true")
	}
	// An empty metrics buffer returns 0 snapshots.
	// The data field may be nil/empty slice — both mean 0 elements.
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(resp.Data))
	}
}

func TestGetSystemHistory_Returns200WithSnapshots(t *testing.T) {
	buf := service.NewMetricsBuffer(120)

	snaps := []model.MetricsSnapshot{
		{Timestamp: "t1", CPUPercent: 10.0, MemoryPercent: 50.0, DiskPercent: 20.0},
		{Timestamp: "t2", CPUPercent: 20.0, MemoryPercent: 60.0, DiskPercent: 25.0},
		{Timestamp: "t3", CPUPercent: 30.0, MemoryPercent: 70.0, DiskPercent: 30.0},
	}
	for _, s := range snaps {
		buf.Push(s)
	}

	h := NewSystemHandler()
	h.SetMetricsBuffer(buf)

	req := httptest.NewRequest(http.MethodGet, "/api/system/history", nil)
	w := httptest.NewRecorder()

	h.GetSystemHistory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp model.ApiResponse[[]model.MetricsSnapshot]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response must be valid JSON: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(resp.Data))
	}

	// Verify chronological order and field content
	if resp.Data[0].CPUPercent != 10.0 {
		t.Errorf("resp.Data[0].CPUPercent = %f, want 10.0", resp.Data[0].CPUPercent)
	}
	if resp.Data[2].CPUPercent != 30.0 {
		t.Errorf("resp.Data[2].CPUPercent = %f, want 30.0", resp.Data[2].CPUPercent)
	}
}

func TestGetSystemHistory_NParsesQueryParam(t *testing.T) {
	buf := service.NewMetricsBuffer(120)

	// Push 10 snapshots
	for i := 1; i <= 10; i++ {
		buf.Push(model.MetricsSnapshot{
			Timestamp:  "t" + string(rune('0'+i)),
			CPUPercent: float64(i),
		})
	}

	h := NewSystemHandler()
	h.SetMetricsBuffer(buf)

	// Request only 3
	req := httptest.NewRequest(http.MethodGet, "/api/system/history?n=3", nil)
	w := httptest.NewRecorder()

	h.GetSystemHistory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp model.ApiResponse[[]model.MetricsSnapshot]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response must be valid JSON: %v", err)
	}

	if len(resp.Data) != 3 {
		t.Fatalf("?n=3 expected 3 snapshots, got %d", len(resp.Data))
	}

	// Most recent 3 are snapshots 8, 9, 10
	if resp.Data[0].CPUPercent != 8.0 {
		t.Errorf("resp.Data[0].CPUPercent = %f, want 8.0 (oldest of last 3)", resp.Data[0].CPUPercent)
	}
	if resp.Data[2].CPUPercent != 10.0 {
		t.Errorf("resp.Data[2].CPUPercent = %f, want 10.0 (newest)", resp.Data[2].CPUPercent)
	}
}

func TestGetSystemHistory_NDefaultsTo120(t *testing.T) {
	buf := service.NewMetricsBuffer(120)

	// Push 5 snapshots — default n=120 should return all 5
	for i := 1; i <= 5; i++ {
		buf.Push(model.MetricsSnapshot{
			Timestamp:  "t" + string(rune('0'+i)),
			CPUPercent: float64(i),
		})
	}

	h := NewSystemHandler()
	h.SetMetricsBuffer(buf)

	req := httptest.NewRequest(http.MethodGet, "/api/system/history", nil)
	w := httptest.NewRecorder()

	h.GetSystemHistory(w, req)

	var resp model.ApiResponse[[]model.MetricsSnapshot]
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response must be valid JSON: %v", err)
	}

	if len(resp.Data) != 5 {
		t.Fatalf("default n=120 expected all 5 snapshots, got %d", len(resp.Data))
	}
}
