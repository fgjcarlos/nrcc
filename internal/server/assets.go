package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"nrcc/internal/middleware"
	"nrcc/internal/model"
	"nrcc/internal/service"
)

// registerAssetRoutes registers branding asset upload, list, delete, and static serving routes.
func registerAssetRoutes(router chi.Router, authService *service.AuthService, assetService *service.AssetService) {
	if assetService == nil {
		return
	}

	// Static serving of assets — public, no auth needed so Node-RED editor can reference them
	router.Get("/assets/{category}/{filename}", func(w http.ResponseWriter, r *http.Request) {
		category := chi.URLParam(r, "category")
		filename := chi.URLParam(r, "filename")

		// Sanitize: no path traversal
		if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
			http.Error(w, "invalid filename", http.StatusBadRequest)
			return
		}

		http.ServeFile(w, r, assetService.AssetsDir()+"/"+category+"/"+filename)
	})

	// API routes for asset management — require auth
	router.Group(func(r chi.Router) {
		if authService != nil {
			r.Use(middleware.RequireAuth(authService))
			r.Use(middleware.RequireCSRF(authService))
			r.Use(middleware.RequireRole(model.RoleAdmin))
		}

		// POST /api/assets/{category}/upload — multipart upload
		r.Post("/api/assets/{category}/upload", func(w http.ResponseWriter, r *http.Request) {
			category := chi.URLParam(r, "category")

			// Limit request body
			r.Body = http.MaxBytesReader(w, r.Body, 2<<20+1024) // 2MB + overhead

			if err := r.ParseMultipartForm(2 << 20); err != nil {
				respondError(w, http.StatusBadRequest, "UPLOAD_TOO_LARGE", "file too large or invalid multipart form")
				return
			}

			file, header, err := r.FormFile("file")
			if err != nil {
				respondError(w, http.StatusBadRequest, "UPLOAD_MISSING_FILE", "missing file in upload")
				return
			}
			defer file.Close()

			asset, err := assetService.Upload(category, file, header)
			if err != nil {
				respondError(w, http.StatusBadRequest, "UPLOAD_FAILED", err.Error())
				return
			}

			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("asset.upload", claims.Username, "uploaded "+category+" asset: "+asset.Original)
				}
			}

			respondOK(w, asset)
		})

		// GET /api/assets/{category} — list assets
		r.Get("/api/assets/{category}", func(w http.ResponseWriter, r *http.Request) {
			category := chi.URLParam(r, "category")

			list, err := assetService.List(category)
			if err != nil {
				respondError(w, http.StatusBadRequest, "ASSETS_LIST_FAILED", err.Error())
				return
			}

			respondOK(w, list)
		})

		// DELETE /api/assets/{category}/{id} — delete asset
		r.Delete("/api/assets/{category}/{id}", func(w http.ResponseWriter, r *http.Request) {
			category := chi.URLParam(r, "category")
			id := chi.URLParam(r, "id")

			if err := assetService.Delete(category, id); err != nil {
				respondError(w, http.StatusBadRequest, "ASSET_DELETE_FAILED", err.Error())
				return
			}

			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("asset.delete", claims.Username, "deleted "+category+" asset: "+id)
				}
			}

			respondOK(w, map[string]any{"deleted": true})
		})
	})
}
