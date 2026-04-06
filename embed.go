package main

import "embed"

// frontend/dist is intentionally committed with a placeholder file so the
// project can compile before the first real frontend build.
//
//go:embed frontend/dist/*
var frontendFS embed.FS
