package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// SPAHandler returns an HTTP handler that serves an SPA from embedded files
func SPAHandler(fsys embed.FS) http.Handler {
	sub, err := fs.Sub(fsys, "frontend/dist")
	if err != nil {
		// Return a handler that serves a placeholder if embed fails
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "api" || strings.HasPrefix(path, "api/") {
				model.RespondError(w, http.StatusNotFound, "API_ROUTE_NOT_FOUND", "API endpoint not found")
				return
			}
			http.Error(w, "SPA files not found", http.StatusNotFound)
		})
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the path
		path := strings.TrimPrefix(r.URL.Path, "/")

		// Try to serve the file as-is
		f, err := sub.Open(path)
		if err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// API typos must never fall through to the SPA index. A 200 HTML
		// response looks successful to browser clients and was the root cause
		// of several controls silently doing nothing.
		if path == "api" || strings.HasPrefix(path, "api/") {
			model.RespondError(w, http.StatusNotFound, "API_ROUTE_NOT_FOUND", "API endpoint not found")
			return
		}

		// Fallback to index.html for SPA routing
		r2 := r.WithContext(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}
