package server

import (
	"embed"
	"log/slog"
	"os"

	"github.com/fgjcarlos/nrcc/internal/middleware"
)

// embedFS is the embedded frontend filesystem (set by main.go)
var embedFS embed.FS

// Config holds server configuration
type Config struct {
	Port       string
	DataDir    string
	JWTSecret  string
	HTTPLogger *slog.Logger
	CORS       middleware.CORSConfig
}

// NewDefaultHTTPLogger returns a fresh server-owned logger for HTTP events.
func NewDefaultHTTPLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// SetEmbedFS sets the embedded filesystem (called from main.go)
func SetEmbedFS(fs embed.FS) {
	embedFS = fs
}
