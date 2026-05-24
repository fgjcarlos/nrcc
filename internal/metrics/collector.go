package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsCollector holds all application-level Prometheus metrics and a custom registry.
type MetricsCollector struct {
	registry         *prometheus.Registry
	processCollector *ProcessCollector

	// LoginAttempts counts login attempts by result label (success, failure).
	LoginAttempts *prometheus.CounterVec
	// BackupCreated counts backups created by type label (manual, auto, pre_restore).
	BackupCreated *prometheus.CounterVec
	// RestoreAttempts counts restore attempts by result label (success, failure).
	RestoreAttempts *prometheus.CounterVec
	// UpdateAttempts counts update attempts by result label (success, failure).
	UpdateAttempts *prometheus.CounterVec
	// LibraryOps counts library operations by operation (install, uninstall) and result (success, failure).
	LibraryOps *prometheus.CounterVec
}

// NewCollector creates a MetricsCollector with a private Prometheus registry
// and registers all application metrics plus Go runtime and process collectors.
func NewCollector() *MetricsCollector {
	reg := prometheus.NewRegistry()

	loginAttempts := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nrcc_login_attempts_total",
			Help: "Total number of login attempts, labeled by result (success or failure).",
		},
		[]string{"result"},
	)

	backupCreated := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nrcc_backup_created_total",
			Help: "Total number of backups created, labeled by type (manual, auto, pre_restore).",
		},
		[]string{"type"},
	)

	restoreAttempts := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nrcc_restore_attempts_total",
			Help: "Total number of restore attempts, labeled by result (success or failure).",
		},
		[]string{"result"},
	)

	updateAttempts := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nrcc_update_attempts_total",
			Help: "Total number of update attempts, labeled by result (success or failure).",
		},
		[]string{"result"},
	)

	libraryOps := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nrcc_library_operations_total",
			Help: "Total number of library operations, labeled by operation (install, uninstall) and result (success, failure).",
		},
		[]string{"operation", "result"},
	)

	procCollector := newProcessCollector()

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		loginAttempts,
		backupCreated,
		restoreAttempts,
		updateAttempts,
		libraryOps,
		procCollector,
	)

	return &MetricsCollector{
		registry:         reg,
		processCollector: procCollector,
		LoginAttempts:    loginAttempts,
		BackupCreated:    backupCreated,
		RestoreAttempts:  restoreAttempts,
		UpdateAttempts:   updateAttempts,
		LibraryOps:       libraryOps,
	}
}

// Handler returns an http.Handler that serves Prometheus metrics from the private registry.
func (mc *MetricsCollector) Handler() http.Handler {
	return promhttp.HandlerFor(mc.registry, promhttp.HandlerOpts{})
}

// SetProcessManager wires a ProcessManagerSource into the ProcessCollector so it can
// read live runtime status when scraping occurs.
func (mc *MetricsCollector) SetProcessManager(pm ProcessManagerSource) {
	mc.processCollector.setSource(pm)
}

// RecordLoginAttempt increments the login attempts counter.
func (mc *MetricsCollector) RecordLoginAttempt(success bool) {
	result := resultLabel(success)
	mc.LoginAttempts.WithLabelValues(result).Inc()
}

// RecordBackupCreated increments the backup created counter with the given backup type.
func (mc *MetricsCollector) RecordBackupCreated(backupType string) {
	mc.BackupCreated.WithLabelValues(backupType).Inc()
}

// RecordRestoreAttempt increments the restore attempts counter.
func (mc *MetricsCollector) RecordRestoreAttempt(success bool) {
	result := resultLabel(success)
	mc.RestoreAttempts.WithLabelValues(result).Inc()
}

// RecordUpdateAttempt increments the update attempts counter.
func (mc *MetricsCollector) RecordUpdateAttempt(success bool) {
	result := resultLabel(success)
	mc.UpdateAttempts.WithLabelValues(result).Inc()
}

// RecordLibraryOperation increments the library operations counter with the given operation and result.
func (mc *MetricsCollector) RecordLibraryOperation(operation string, success bool) {
	result := resultLabel(success)
	mc.LibraryOps.WithLabelValues(operation, result).Inc()
}

// resultLabel converts a boolean success flag to a Prometheus label string.
func resultLabel(success bool) string {
	if success {
		return "success"
	}
	return "failure"
}
