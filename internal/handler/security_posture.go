package handler

import (
	"net/http"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

type securityPostureSnapshotter interface {
	Snapshot(time.Time) (service.SecurityPosture, error)
}

// SecurityPostureHandler serves the admin-only posture snapshot endpoint.
type SecurityPostureHandler struct {
	service securityPostureSnapshotter
}

// NewSecurityPostureHandler creates the HTTP adapter for posture snapshots.
func NewSecurityPostureHandler(service securityPostureSnapshotter) *SecurityPostureHandler {
	return &SecurityPostureHandler{service: service}
}

// Get handles GET /api/system/security-posture.
func (h *SecurityPostureHandler) Get(w http.ResponseWriter, _ *http.Request) {
	posture, err := h.service.Snapshot(time.Now())
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "SECURITY_POSTURE_ERROR", "Failed to read security posture")
		return
	}
	model.RespondJSON(w, http.StatusOK, posture)
}
