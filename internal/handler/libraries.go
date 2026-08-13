package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

// Per-handler hard timeouts. Each is bounded above the service-level constant
// so the handler timeout is the FIRST cap; the service's installTimeout /
// searchHTTPTimeout is the fallback. Regression for HIGH-004 + HIGH-013: a
// client disconnect (request ctx cancel) must abort the subprocess promptly.
const (
	installHandlerTimeout   = 5 * time.Minute
	uninstallHandlerTimeout = 5 * time.Minute
	searchHandlerTimeout    = 35 * time.Second
	checkHandlerTimeout     = 12 * time.Second
)

// libraryMetricsRecorder is the narrow interface for recording library operation metrics.
// Using an interface instead of *metrics.MetricsCollector keeps LibraryHandler
// testable with simple stubs and avoids a direct dependency on the metrics package.
type libraryMetricsRecorder interface {
	RecordLibraryOperation(operation string, success bool)
}

// LibraryHandler handles library/npm package endpoints
type LibraryHandler struct {
	svc            *service.LibraryService
	libraryMetrics libraryMetricsRecorder
}

// NewLibraryHandler creates a new library handler
func NewLibraryHandler(svc *service.LibraryService) *LibraryHandler {
	return &LibraryHandler{svc: svc}
}

// SetLibraryMetrics injects the metrics recorder for library operations.
func (h *LibraryHandler) SetLibraryMetrics(m libraryMetricsRecorder) { h.libraryMetrics = m }

// GetLibraries lists installed packages
// GET /api/libraries
func (h *LibraryHandler) GetLibraries(w http.ResponseWriter, r *http.Request) {
	libs, err := h.svc.List()
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "LIBRARY_ERROR", err.Error())
		return
	}

	if libs == nil {
		libs = []model.LibraryInfo{}
	}

	model.RespondJSON(w, http.StatusOK, libs)
}

// PostInstall installs a package
// POST /api/libraries/install
func (h *LibraryHandler) PostInstall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if !DecodeJSON(w, r, &req) {
		return
	}

	if req.Name == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Package name is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), installHandlerTimeout)
	defer cancel()

	err := h.svc.Install(ctx, req.Name)
	if err != nil {
		if h.libraryMetrics != nil {
			h.libraryMetrics.RecordLibraryOperation("install", false)
		}
		if errors.Is(err, service.ErrInvalidPackageName) {
			model.RespondError(w, http.StatusBadRequest, "INVALID_PACKAGE_NAME", err.Error())
			return
		}
		model.RespondError(w, http.StatusInternalServerError, "INSTALL_ERROR", err.Error())
		return
	}

	if h.libraryMetrics != nil {
		h.libraryMetrics.RecordLibraryOperation("install", true)
	}
	model.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Package installed",
	})
}

// DeleteLibrary uninstalls a package
// DELETE /api/libraries/{name}
func (h *LibraryHandler) DeleteLibrary(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	ctx, cancel := context.WithTimeout(r.Context(), uninstallHandlerTimeout)
	defer cancel()

	err := h.svc.Uninstall(ctx, name)
	if err != nil {
		if h.libraryMetrics != nil {
			h.libraryMetrics.RecordLibraryOperation("uninstall", false)
		}
		if errors.Is(err, service.ErrInvalidPackageName) {
			model.RespondError(w, http.StatusBadRequest, "INVALID_PACKAGE_NAME", err.Error())
			return
		}
		model.RespondError(w, http.StatusInternalServerError, "UNINSTALL_ERROR", err.Error())
		return
	}

	if h.libraryMetrics != nil {
		h.libraryMetrics.RecordLibraryOperation("uninstall", true)
	}
	w.WriteHeader(http.StatusNoContent)
}

// PostSearch searches npm registry
// POST /api/libraries/search
func (h *LibraryHandler) PostSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}

	if !DecodeJSON(w, r, &req) {
		return
	}

	if req.Query == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Query is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), searchHandlerTimeout)
	defer cancel()

	results, err := h.svc.Search(ctx, req.Query)
	if err != nil {
		// Search is an external registry convenience feature. Do not turn registry
		// failures into UI-breaking 500s; return an empty result set instead.
		model.RespondJSON(w, http.StatusOK, []model.LibraryInfo{})
		return
	}

	if results == nil {
		results = []model.LibraryInfo{}
	}

	model.RespondJSON(w, http.StatusOK, results)
}

// GetLibraryCheck checks if a package is available
// GET /api/libraries/{name}/check
func (h *LibraryHandler) GetLibraryCheck(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	ctx, cancel := context.WithTimeout(r.Context(), checkHandlerTimeout)
	defer cancel()

	available, err := h.svc.Check(ctx, name)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "CHECK_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"name":      name,
		"available": available,
	})
}
