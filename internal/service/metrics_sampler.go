package service

import (
	"context"
	"sync"
	"time"
)

// MetricsSampler periodically samples host metrics and stores them in a MetricsBuffer.
type MetricsSampler struct {
	buffer   *MetricsBuffer
	interval time.Duration
	lastCPU  float64
	mu       sync.Mutex
}

// NewMetricsSampler creates a MetricsSampler that writes into buf at the given interval.
func NewMetricsSampler(buf *MetricsBuffer, interval time.Duration) *MetricsSampler {
	return &MetricsSampler{
		buffer:   buf,
		interval: interval,
	}
}

// Start begins periodic sampling. It collects an initial sample immediately, then
// repeats on every tick. It returns when ctx is cancelled.
func (ms *MetricsSampler) Start(ctx context.Context) {
	ms.sample()

	ticker := time.NewTicker(ms.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ms.sample()
		}
	}
}

// LastCPU returns the CPU percentage captured by the most recent sample.
func (ms *MetricsSampler) LastCPU() float64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.lastCPU
}

// sample collects one snapshot and pushes it to the buffer.
func (ms *MetricsSampler) sample() {
	snap := sampleHost()
	snap.Timestamp = time.Now().UTC().Format(time.RFC3339)
	ms.buffer.Push(snap)

	ms.mu.Lock()
	ms.lastCPU = snap.CPUPercent
	ms.mu.Unlock()
}

// sampleHost is implemented in platform-specific files:
//   - metrics_sampler_linux.go  (build tag: linux)
//   - metrics_sampler_other.go  (build tag: !linux)
