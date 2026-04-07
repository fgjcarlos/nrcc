package model

import "time"

// DoctorCheck represents a single diagnostic check
type DoctorCheck struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Status   string `json:"status"`   // pass, warn, fail
	Severity string `json:"severity"` // critical, warning
	Message  string `json:"message"`
}

// Check statuses
const (
	CheckStatusPass = "pass"
	CheckStatusWarn = "warn"
	CheckStatusFail = "fail"
)

// Check severities
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
)

// DoctorReport represents the overall diagnostic report
type DoctorReport struct {
	GeneratedAt   time.Time     `json:"generatedAt"`
	OverallStatus string        `json:"overallStatus"` // healthy, degraded, critical
	Checks        []DoctorCheck `json:"checks"`
}

// Overall statuses
const (
	OverallHealthy  = "healthy"
	OverallDegraded = "degraded"
	OverallCritical = "critical"
)
