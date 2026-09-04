package service

// ResourceMetrics is a consistent snapshot of the resources available to NRCC.
// Values are scoped either to the host or to the container that runs NRCC.
type ResourceMetrics struct {
	Scope string

	CPUAvailable    bool
	CPUUsage        float64
	CPUCores        int
	MemoryAvailable bool
	MemoryTotal     uint64
	MemoryUsed      uint64
	MemoryFree      uint64
	DiskAvailable   bool
	DiskTotal       uint64
	DiskUsed        uint64
	DiskFree        uint64
}
