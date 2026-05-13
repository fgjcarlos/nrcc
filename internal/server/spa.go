package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// SPAHandler returns an HTTP handler that serves an SPA from embedded files
func SPAHandler(fsys embed.FS) http.Handler {
	sub, err := fs.Sub(fsys, "frontend/dist")
	if err != nil {
		// Return a handler that serves a placeholder if embed fails
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA routing
		r2 := r.WithContext(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}
