package service

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/composedof2/nrcc/internal/model"
)

func TestMetricsBuffer_Push_And_Recent(t *testing.T) {
	buf := NewMetricsBuffer(5)

	snap := model.MetricsSnapshot{
		Timestamp:     "2024-01-01T00:00:00Z",
		CPUPercent:    10.5,
		MemoryPercent: 20.0,
		DiskPercent:   30.0,
	}
	buf.Push(snap)

	recent := buf.Recent(1)
	if len(recent) != 1 {
		t.Fatalf("Recent(1) len = %d, want 1", len(recent))
	}
	if recent[0].CPUPercent != snap.CPUPercent {
		t.Errorf("CPUPercent = %f, want %f", recent[0].CPUPercent, snap.CPUPercent)
	}
}

func TestMetricsBuffer_Recent_ReturnsChronologicalOrder(t *testing.T) {
	buf := NewMetricsBuffer(5)

	snaps := []model.MetricsSnapshot{
		{Timestamp: "t1", CPUPercent: 10.0},
		{Timestamp: "t2", CPUPercent: 20.0},
		{Timestamp: "t3", CPUPercent: 30.0},
	}
	for _, s := range snaps {
		buf.Push(s)
	}

	recent := buf.Recent(3)
	if len(recent) != 3 {
		t.Fatalf("Recent(3) len = %d, want 3", len(recent))
	}
	// oldest first
	if recent[0].CPUPercent != 10.0 {
		t.Errorf("recent[0].CPUPercent = %f, want 10.0", recent[0].CPUPercent)
	}
	if recent[2].CPUPercent != 30.0 {
		t.Errorf("recent[2].CPUPercent = %f, want 30.0", recent[2].CPUPercent)
	}
}

func TestMetricsBuffer_Recent_LimitedByCount(t *testing.T) {
	buf := NewMetricsBuffer(5)

	buf.Push(model.MetricsSnapshot{Timestamp: "t1", CPUPercent: 1.0})

	recent := buf.Recent(10) // ask for more than available
	if len(recent) != 1 {
		t.Fatalf("Recent(10) len = %d, want 1 (only 1 pushed)", len(recent))
	}
}

func TestMetricsBuffer_Last_EmptyBuffer(t *testing.T) {
	buf := NewMetricsBuffer(5)

	_, ok := buf.Last()
	if ok {
		t.Error("Last() ok = true on empty buffer, want false")
	}
}

func TestMetricsBuffer_Last_ReturnsNewest(t *testing.T) {
	buf := NewMetricsBuffer(5)

	buf.Push(model.MetricsSnapshot{Timestamp: "t1", CPUPercent: 5.0})
	buf.Push(model.MetricsSnapshot{Timestamp: "t2", CPUPercent: 99.0})

	snap, ok := buf.Last()
	if !ok {
		t.Fatal("Last() ok = false, want true after pushes")
	}
	if snap.CPUPercent != 99.0 {
		t.Errorf("Last().CPUPercent = %f, want 99.0", snap.CPUPercent)
	}
}

func TestMetricsBuffer_RingOverwrite(t *testing.T) {
	buf := NewMetricsBuffer(3)

	// Push 4 entries into capacity-3 buffer — oldest must be dropped
	for i := 1; i <= 4; i++ {
		buf.Push(model.MetricsSnapshot{Timestamp: fmt.Sprintf("t%d", i), CPUPercent: float64(i)})
	}

	recent := buf.Recent(3)
	if len(recent) != 3 {
		t.Fatalf("Recent(3) len = %d, want 3", len(recent))
	}
	// Entries 2,3,4 should remain
	if recent[0].CPUPercent != 2.0 {
		t.Errorf("recent[0].CPUPercent = %f, want 2.0", recent[0].CPUPercent)
	}
	if recent[2].CPUPercent != 4.0 {
		t.Errorf("recent[2].CPUPercent = %f, want 4.0", recent[2].CPUPercent)
	}
}

func TestMetricsBuffer_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()

	buf := NewMetricsBuffer(10)
	const goroutines = 8
	const ops = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// concurrent writers
	for g := 0; g < goroutines; g++ {
		g := g
		go func() {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				buf.Push(model.MetricsSnapshot{
					Timestamp:  fmt.Sprintf("w%d-i%d", g, i),
					CPUPercent: float64(g*ops + i),
				})
			}
		}()
	}

	// concurrent readers
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				_ = buf.Recent(5)
				_, _ = buf.Last()
				time.Sleep(0) // yield to scheduler
			}
		}()
	}

	wg.Wait() // no race detected by -race flag
}
