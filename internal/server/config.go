package server

import "embed"

// embedFS is the embedded frontend filesystem (set by main.go)
var embedFS embed.FS

// Config holds server configuration
type Config struct {
	Port      string
	DataDir   string
	JWTSecret string
}

// SetEmbedFS sets the embedded filesystem (called from main.go)
func SetEmbedFS(fs embed.FS) {
	embedFS = fs
}
