package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"

	"nrcc/internal/middleware"
)

// registerDiagnosticsRoutes registers the diagnostics API endpoints
func registerDiagnosticsRoutes(r chi.Router, cfg Config) {
	if cfg.Auth == nil || cfg.Logs == nil || cfg.Jobs == nil {
		// Diagnostics require auth, logs, and jobs services
		return
	}

	r.Route("/api/diagnostics", func(r chi.Router) {
		// Require authentication for all diagnostics endpoints
		r.Use(middleware.RequireAuth(cfg.Auth))
		r.Use(middleware.RequireCSRF(cfg.Auth))

		// GET /api/diagnostics/report - run doctor checks
		r.Get("/report", func(w http.ResponseWriter, req *http.Request) {
			if cfg.Doctor == nil {
				respondError(w, http.StatusServiceUnavailable, "DOCTOR_UNAVAILABLE", "doctor service not available")
				return
			}

			report := cfg.Doctor.Run(req.Context())
			respondOK(w, report)
		})

		// GET /api/diagnostics/logs - query logs with pagination
		r.Get("/logs", func(w http.ResponseWriter, req *http.Request) {
			level := req.URL.Query().Get("level")
			source := req.URL.Query().Get("source")
			limitStr := req.URL.Query().Get("limit")
			offsetStr := req.URL.Query().Get("offset")

			limit := 100
			offset := 0

			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
				limit = l
			}
			if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
				offset = o
			}

			logs := cfg.Logs.Get(limit, level, source)

			respondOK(w, map[string]interface{}{
				"logs":   logs,
				"limit":  limit,
				"offset": offset,
			})
		})

		// GET /api/diagnostics/jobs - query job history with pagination
		r.Get("/jobs", func(w http.ResponseWriter, req *http.Request) {
			jobType := req.URL.Query().Get("type")
			status := req.URL.Query().Get("status")
			limitStr := req.URL.Query().Get("limit")
			offsetStr := req.URL.Query().Get("offset")

			limit := 50
			offset := 0

			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
				limit = l
			}
			if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
				offset = o
			}

			jobs, err := cfg.Jobs.Get(limit, offset, jobType, status)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "JOBS_QUERY_FAILED", err.Error())
				return
			}

			respondOK(w, map[string]interface{}{
				"jobs":   jobs,
				"limit":  limit,
				"offset": offset,
			})
		})

		// POST /api/diagnostics/export - generate support bundle
		r.Post("/export", func(w http.ResponseWriter, req *http.Request) {
			if cfg.Support == nil {
				respondError(w, http.StatusServiceUnavailable, "SUPPORT_UNAVAILABLE", "support bundle service not available")
				return
			}

			path, err := cfg.Support.Export(req.Context())
			if err != nil {
				respondError(w, http.StatusInternalServerError, "EXPORT_FAILED", fmt.Sprintf("failed to generate support bundle: %v", err))
				return
			}

			// Get file size
			var size int64
			if f, err := os.Stat(path); err == nil {
				size = f.Size()
			}

			respondOK(w, map[string]interface{}{
				"path": path,
				"size": size,
			})
		})
	})
}
