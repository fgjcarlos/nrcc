package service

import (
	"context"
	"testing"
	"time"
)

func TestMetricsSampler_Start_PopulatesBuffer(t *testing.T) {
	t.Parallel()

	buf := NewMetricsBuffer(10)
	sampler := NewMetricsSampler(buf, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sampler.Start(ctx)

	// Wait for at least one sample to land
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		_, ok := buf.Last()
		if ok {
			return // buffer was populated — test passes
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("buffer was not populated within 500ms after Start()")
}

func TestMetricsSampler_Start_StopsOnContextCancel(t *testing.T) {
	t.Parallel()

	buf := NewMetricsBuffer(10)
	sampler := NewMetricsSampler(buf, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		sampler.Start(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// goroutine exited cleanly
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Start() did not return within 500ms after context cancel")
	}
}

func TestMetricsSampler_LastCPU_ReturnsCachedValue(t *testing.T) {
	t.Parallel()

	buf := NewMetricsBuffer(10)
	sampler := NewMetricsSampler(buf, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sampler.Start(ctx)

	// Wait until the sampler has run at least once
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		_, ok := buf.Last()
		if ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	cpu := sampler.LastCPU()
	// CPU percent must be in [0, 100]
	if cpu < 0 || cpu > 100 {
		t.Errorf("LastCPU() = %f, want value in [0, 100]", cpu)
	}
}

func TestMetricsSampler_MultipleSamples(t *testing.T) {
	t.Parallel()

	buf := NewMetricsBuffer(10)
	sampler := NewMetricsSampler(buf, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sampler.Start(ctx)

	// Wait for at least 3 samples
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		recent := buf.Recent(3)
		if len(recent) >= 3 {
			return // collected enough samples
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("did not collect 3 samples within 1s; got %d", len(buf.Recent(10)))
}
