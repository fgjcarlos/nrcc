package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
)

func registerSPARoutes(router chi.Router, frontend fs.FS) {
	if frontend == nil {
		router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
		return
	}

	distFS, err := fs.Sub(frontend, "frontend/dist")
	if err != nil {
		router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
		return
	}

	fileServer := http.FileServer(http.FS(distFS))
	router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		requestPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requestPath == "." || requestPath == "" {
			requestPath = "index.html"
		}

		if _, err := fs.Stat(distFS, requestPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
