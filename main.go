package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/server"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
	"github.com/fgjcarlos/nrcc/internal/ui"
)

var Version string = "dev"

func main() {
	// Initialize UI (pterm configuration)
	ui.Init()

	// The nrcc binary is server-only (ADR 0003 / Docker-first).
	runServer()
}

func runServer() {
	// Initialize UI (pterm configuration)
	ui.Init()

	// Load environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	jwtSecret, err := resolveJWTSecret(dataDir)
	if err != nil {
		log.Fatalf("JWT secret error: %v", err)
	}

	// Print startup banner
	ui.Banner(Version, port, dataDir)

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize JSON stores
	usersPath := filepath.Join(dataDir, "cc-users.json")
	usersStore := store.NewJSONStore[model.CCUsers](usersPath)
	sessionsPath := filepath.Join(dataDir, "refresh_sessions.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](sessionsPath)

	// Initialize auth service
	authSvc := service.NewAuthService(jwtSecret, usersStore, sessionStore)

	// Initialize log buffer (1000 max entries)
	logBuffer := service.NewLogBuffer(1000)

	hostSvc := service.NewHostService(dataDir)

	ui.SectionHeader("Bootstrap")
	if err := hostSvc.BootstrapCLI(); err != nil {
		ui.Infof("Bootstrap warning: %v", err)
	}

	// Initialize process manager
	nodeRedCmd := os.Getenv("NODE_RED_CMD")
	if nodeRedCmd == "" {
		nodeRedCmd = "node-red"
	}
	pm := service.NewProcessManager(nodeRedCmd, dataDir, logBuffer)
	configSvc := service.NewConfigServiceWithHost(dataDir, hostSvc)
	pm.SetEnvService(service.NewEnvService(configSvc, os.Getenv("NRCC_ENCRYPTION_KEY")))

	// Node-RED managed mode is ON by default.
	// Set NRCC_MANAGE_NODE_RED=false to disable.
	ui.SectionHeader("Node-RED Startup")
	manageRuntime := os.Getenv("NRCC_MANAGE_NODE_RED") != "false"
	if manageRuntime {
		spinner := ui.StartSpinner("Starting Node-RED…")
		if err := pm.Start(); err != nil {
			spinner.Fail(fmt.Sprintf("Failed to start Node-RED: %v", err))
		} else if pm.IsExternalMode() {
			spinner.Success("Attached to existing Node-RED — http://localhost:1880")
		} else {
			spinner.Success("Node-RED started — http://localhost:1880")
		}
	}

	// Parse CORS configuration from environment
	corsCfg := middleware.CORSConfig{
		AllowUnsafeWildcard: os.Getenv("NRCC_CORS_UNSAFE_WILDCARD") == "true",
	}
	if origins := os.Getenv("NRCC_CORS_ORIGINS"); origins != "" {
		for _, o := range strings.Split(origins, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				corsCfg.AllowedOrigins = append(corsCfg.AllowedOrigins, trimmed)
			}
		}
	}

	// Set embedded filesystem in server package
	server.SetEmbedFS(frontendFS)

	// Create and configure server
	srv := server.NewServerWithConfig(authSvc, dataDir, corsCfg)
	if manageRuntime {
		srv.SetProcessManager(pm)
	}
	srv.SetLogBuffer(logBuffer)

	// Create HTTP server
	ui.SectionHeader("Server Starting")
	httpSrv := &http.Server{
		Addr:         ":" + port,
		Handler:      srv,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		ui.Info(fmt.Sprintf("Server listening on %s", httpSrv.Addr))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ui.Error(fmt.Sprintf("Server error: %v", err))
		}
	}()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
		// OS signal received (SIGINT/SIGTERM)
	case <-srv.GetShutdownChannel():
		// Handler-triggered shutdown (e.g., Docker restart/stop)
	}

	ui.SectionHeader("Shutting Down")

	// Stop Node-RED process first
	if manageRuntime {
		if err := pm.Stop(); err != nil {
			ui.Error(fmt.Sprintf("Node-RED shutdown error: %v", err))
		} else {
			ui.Info("Node-RED stopped")
		}
	}

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		ui.Error(fmt.Sprintf("Server shutdown error: %v", err))
	}

	ui.Info("Shutdown complete")
}

var placeholderSecrets = []string{
	"cc-secret-change-in-production",
	"change-me-in-production",
	"dev-secret-not-for-production",
}

func resolveJWTSecret(dataDir string) (string, error) {
	secret := os.Getenv("JWT_SECRET")

	if secret != "" {
		for _, placeholder := range placeholderSecrets {
			if strings.EqualFold(secret, placeholder) {
				return "", fmt.Errorf(
					"JWT_SECRET is set to a known placeholder (%q) — provide a real secret",
					secret,
				)
			}
		}
		return secret, nil
	}

	// No env var: generate and persist a local secret.
	secretPath := filepath.Join(dataDir, "jwt_secret")
	if data, err := os.ReadFile(secretPath); err == nil {
		if s := strings.TrimSpace(string(data)); len(s) >= 32 {
			ui.Info("Using persisted JWT secret from " + secretPath)
			return s, nil
		}
	}

	generated, err := service.GenerateJWTSecret()
	if err != nil {
		return "", fmt.Errorf("failed to auto-generate JWT secret: %w", err)
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(secretPath, []byte(generated+"\n"), 0600); err != nil {
		return "", fmt.Errorf("failed to persist JWT secret: %w", err)
	}

	ui.Warn("JWT_SECRET not set — generated a random secret at " + secretPath)
	return generated, nil
}
