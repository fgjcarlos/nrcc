//go:build !linux

package service

// CollectResourceMetrics is unavailable outside Linux because NRCC has no
// portable way to establish an equivalent host or container resource scope.
func CollectResourceMetrics() ResourceMetrics {
	return ResourceMetrics{Scope: "unavailable"}
}
