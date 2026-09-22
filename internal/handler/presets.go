package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// PresetsHandler exposes the curated-preset catalogue to the
// frontend. It serves two endpoints:
//
//   - GET  /api/presets                  — list curated presets
//   - POST /api/presets/{id}/preview     — preview a preset apply
//
// Apply-to-disk is intentionally NOT in this handler: the operator
// reviews the redacted preview and then confirms by writing through
// the existing /api/settings/raw endpoint, so the settings.js write
// path stays single-flight and audited (slice 2 of #758). Slice 3
// of #764 only adds the catalogue + preview surface; the actual
// write path reuses the existing /api/settings/raw handler.
//
// Authorization (admin role) is enforced by middleware.RequireAdmin
// on the route; this handler trusts the request context to contain
// claims and never makes the admin/non-admin decision itself.
type PresetsHandler struct {
	registry *service.PresetRegistry
}

// NewPresetsHandler returns a handler backed by the default preset
// registry (the four core presets shipped with slice 1).
func NewPresetsHandler() *PresetsHandler {
	return &PresetsHandler{registry: service.NewPresetRegistry()}
}

// presetView is the JSON shape returned to the frontend. It is a
// subset of service.Preset: the registry carries BuildEdits closures
// that must not be serialised, and ManagedKeys is implicit from the
// surfaces contract documented in slice 1.
type presetView struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Surfaces        []string `json:"surfaces"`
	Trust           string   `json:"trust"`
	ChannelBoundary string   `json:"channelBoundary"`
	ManagedKeys     []string `json:"managedKeys"`
}

// presetPreviewRequest is the body of POST /api/presets/{id}/preview.
// Content is the live settings.js source the operator wants to
// preview the preset against.
type presetPreviewRequest struct {
	Content string `json:"content"`
}

// presetPreviewResponse carries the redacted diff the UI renders.
// After is the patched source the operator can confirm by writing
// through /api/settings/raw; Preview is the redacted unified diff
// suitable for direct display and audit log inclusion.
type presetPreviewResponse struct {
	After    string   `json:"after"`
	Preview  string   `json:"preview"`
	Replaced []string `json:"replaced"`
	Inserted []string `json:"inserted"`
}

// List handles GET /api/presets. Returns the curated catalogue in
// stable ID-sorted order so the UI can render a deterministic menu.
func (h *PresetsHandler) List(w http.ResponseWriter, r *http.Request) {
	presets := h.registry.List()
	out := make([]presetView, 0, len(presets))
	for _, p := range presets {
		surfaces := make([]string, 0, len(p.Surfaces))
		for _, s := range p.Surfaces {
			surfaces = append(surfaces, string(s))
		}
		out = append(out, presetView{
			ID:              p.ID,
			Name:            p.Name,
			Description:     p.Description,
			Surfaces:        surfaces,
			Trust:           string(p.Trust),
			ChannelBoundary: p.ChannelBoundary,
			ManagedKeys:     append([]string(nil), p.ManagedKeys...),
		})
	}
	model.RespondJSON(w, http.StatusOK, out)
}

// Preview handles POST /api/presets/{id}/preview. It runs the
// preset apply against the supplied content WITHOUT touching disk
// and returns the redacted diff + the patched source. The operator
// can review the preview and then confirm by writing through the
// existing /api/settings/raw endpoint.
//
// Failures are surfaced with typed status codes so the UI can
// distinguish "preset not registered" (404) from "malformed request"
// (400) from "source patch rejected" (422).
func (h *PresetsHandler) Preview(w http.ResponseWriter, r *http.Request, presetID string) {
	var req presetPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	res, err := h.registry.ApplyPreset(req.Content, presetID, nil)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnknownPreset):
			model.RespondError(w, http.StatusNotFound, "PRESET_NOT_REGISTERED", err.Error())
		case errors.Is(err, service.ErrSourceNotExports):
			model.RespondError(w, http.StatusUnprocessableEntity, "SOURCE_NOT_EXPORTS", err.Error())
		default:
			model.RespondError(w, http.StatusInternalServerError, "PRESET_PREVIEW_FAILED", err.Error())
		}
		return
	}

	model.RespondJSON(w, http.StatusOK, presetPreviewResponse{
		After:    res.After,
		Preview:  res.Preview,
		Replaced: res.Replaced,
		Inserted: res.Inserted,
	})
}
