package handler

import (
	"encoding/json"
	"net/http"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

// LibraryHandler handles library/npm package endpoints
type LibraryHandler struct {
	svc *service.LibraryService
}

// NewLibraryHandler creates a new library handler
func NewLibraryHandler(svc *service.LibraryService) *LibraryHandler {
	return &LibraryHandler{svc: svc}
}

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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.Name == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Package name is required")
		return
	}

	err := h.svc.Install(req.Name)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "INSTALL_ERROR", err.Error())
		return
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

	err := h.svc.Uninstall(name)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "UNINSTALL_ERROR", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PostSearch searches npm registry
// POST /api/libraries/search
func (h *LibraryHandler) PostSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.Query == "" {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Query is required")
		return
	}

	results, err := h.svc.Search(req.Query)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "SEARCH_ERROR", err.Error())
		return
	}

	if results == nil {
		results = []interface{}{}
	}

	model.RespondJSON(w, http.StatusOK, results)
}

// GetLibraryCheck checks if a package is available
// GET /api/libraries/{name}/check
func (h *LibraryHandler) GetLibraryCheck(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	available, err := h.svc.Check(name)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "CHECK_ERROR", err.Error())
		return
	}

	model.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"name":      name,
		"available": available,
	})
}
