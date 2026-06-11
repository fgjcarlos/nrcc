//go:build !linux

package service

import "github.com/fgjcarlos/nrcc/internal/model"

// sampleHost returns a zero MetricsSnapshot on non-Linux platforms.
func sampleHost() model.MetricsSnapshot {
	return model.MetricsSnapshot{}
}
