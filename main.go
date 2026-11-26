package main

import (
	"context"
	"errors"
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

	if _, err := service.ReadinessTimeoutFromEnv(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Refuse incompatible data before startup creates or recovers any files.
	if err := store.CheckSchemaVersions(dataDir); err != nil {
		log.Fatalf("Cannot start: %v", err)
	}

	jwtSecret, err := resolveJWTSecret(dataDir)
	if err != nil {
		log.Fatalf("JWT secret error: %v", err)
	}

	// Print startup banner
	ui.Banner(Version, port, dataDir)

	// Create data directory if it doesn't exist
	// #nosec G703 -- dataDir is an operator-supplied flag (--data-dir) or env var; not request-derived.
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize JSON stores
	usersPath := filepath.Join(dataDir, "cc-users.json")
	usersStore := store.NewVersionedJSONStore[model.CCUsers](usersPath)
	sessionsPath := filepath.Join(dataDir, "refresh_sessions.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](sessionsPath)

	// Initialize auth service
	authSvc := service.NewAuthService(jwtSecret, usersStore, sessionStore)

	hostSvc := service.NewHostService(dataDir)

	// docker-first (ADR 0003): no bootstrap step. `docker compose up` is
	// the bootstrap; HostService.Detect() powers the /bootstrap read-only
	// status page once the stack is up.

	// Initialize process manager
	nodeRedCmd := os.Getenv("NODE_RED_CMD")
	if nodeRedCmd == "" {
		nodeRedCmd = "node-red"
	}
	pm := service.NewProcessManager(nodeRedCmd, dataDir)
	configSvc := service.NewConfigServiceWithHost(dataDir, hostSvc)
	pm.SetEnvService(service.NewEnvService(configSvc, os.Getenv("NRCC_ENCRYPTION_KEY")))

	// Roll back an apply a previous run never confirmed ready (issue #1012),
	// before any Node-RED instance loads the unverified settings.js.
	settingsPath := hostSvc.Detect().Settings.Path
	restored, err := configSvc.RecoverInterruptedApply(context.Background(), settingsPath)
	if restored != "" {
		ui.Warn("Interrupted settings apply rolled back: restored " + restored)
	}
	if err != nil {
		ui.Error(fmt.Sprintf("Interrupted apply recovery: %v", err))
	}

	// Roll back a restore that died while publishing its staged tree
	// (issue #1024), for the same reason: dataDir must never be read as a
	// mix of the old and the new set.
	if rolledBack, err := service.RecoverInterruptedRestore(dataDir); rolledBack != "" {
		ui.Warn("Interrupted backup restore rolled back: " + rolledBack)
	} else if err != nil {
		ui.Error(fmt.Sprintf("Interrupted restore recovery: %v", err))
	}

	// Recovery can replace live files with a previous snapshot.
	if err := store.CheckSchemaVersions(dataDir); err != nil {
		log.Fatalf("Cannot start after recovery: %v", err)
	}

	// Node-RED managed mode is ON by default.
	// Set NRCC_MANAGE_NODE_RED=false to disable.
	ui.SectionHeader("Node-RED Startup")
	manageRuntime := os.Getenv("NRCC_MANAGE_NODE_RED") != "false"
	recoverInterruptedExternalApply(restored, manageRuntime, func() {
		if err := server.RecoverExternalApplyRuntime(context.Background(), hostSvc, configSvc, dataDir, settingsPath, restored); err != nil {
			ui.Warn("Interrupted apply restored " + restored + "; " + err.Error())
		}
	})
	if manageRuntime {
		spinner := ui.StartSpinner("Starting Node-RED…")
		if err := pm.Start(); err != nil {
			msg := fmt.Sprintf("Failed to start Node-RED: %v", err)
			if errors.Is(err, service.ErrForeignListener) {
				msg += " (set NRCC_MANAGE_NODE_RED=false to run beside a Node-RED that NRCC does not manage)"
			}
			spinner.Fail(msg)
		} else {
			spinner.Success("Node-RED started — http://localhost:1880")
		}
	}

	corsCfg := corsConfigFromEnv()

	// Set embedded filesystem in server package
	server.SetEmbedFS(frontendFS)

	// Create and configure server
	srv := server.NewServerWithConfig(authSvc, server.Config{
		DataDir:    dataDir,
		CORS:       corsCfg,
		HTTPLogger: server.NewDefaultHTTPLogger(),
	})
	if manageRuntime {
		srv.SetProcessManager(pm)
	}

	// Create HTTP server
	ui.SectionHeader("Server Starting")
	httpSrv := &http.Server{
		Addr:              ":" + port,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		// npm installs and safe restores are intentionally synchronous so
		// callers learn whether Node-RED actually reloaded. Their handlers
		// cap work below this server-level deadline.
		WriteTimeout: 11 * time.Minute,
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

	stopRuntime := func() {
		if !manageRuntime {
			return
		}
		if err := pm.Stop(context.Background()); err != nil {
			ui.Error(fmt.Sprintf("Node-RED shutdown error: %v", err))
		} else {
			ui.Info("Node-RED stopped")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := shutdownServer(ctx, httpSrv, srv.Shutdown, stopRuntime); err != nil {
		ui.Error(fmt.Sprintf("Server shutdown error: %v", err))
	}
	ui.Info("Shutdown complete")
}

func recoverInterruptedExternalApply(restored string, manageRuntime bool, recover func()) {
	if restored != "" && !manageRuntime && recover != nil {
		recover()
	}
}

func shutdownServer(ctx context.Context, httpSrv *http.Server, stopServices, stopRuntime func()) error {
	err := httpSrv.Shutdown(ctx)
	stopServices()
	stopRuntime()
	return err
}

func resolveJWTSecret(dataDir string) (string, error) {
	secret := os.Getenv("JWT_SECRET")

	if secret != "" {
		if err := service.ValidateSecret("JWT_SECRET", secret); err != nil {
			return "", err
		}
		return secret, nil
	}

	// No env var: generate and persist a local secret.
	// #nosec G703 -- secretPath is built from operator-supplied dataDir + a constant filename; not request-derived.
	secretPath := filepath.Join(dataDir, "jwt_secret")
	// #nosec G304 -- secretPath is built from operator-supplied dataDir + a constant filename; not request-derived.
	if data, err := os.ReadFile(secretPath); err == nil { //nolint:gosec // G304 G703 operator-supplied dataDir + constant filename
		if s := strings.TrimSpace(string(data)); len(s) >= 32 {
			ui.Info("Using persisted JWT secret from " + secretPath)
			return s, nil
		}
	}

	generated, err := service.GenerateJWTSecret()
	if err != nil {
		return "", fmt.Errorf("failed to auto-generate JWT secret: %w", err)
	}

	// #nosec G703 -- dataDir is an operator-supplied flag (--data-dir) or env var; not request-derived.
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return "", err
	}
	// #nosec G703 -- secretPath is built from operator-supplied dataDir + a constant filename; not derived from request input.
	if err := os.WriteFile(secretPath, []byte(generated+"\n"), 0600); err != nil {
		return "", fmt.Errorf("failed to persist JWT secret: %w", err)
	}

	ui.Warn("JWT_SECRET not set — generated a random secret at " + secretPath)
	return generated, nil
}

// corsConfigFromEnv reads the CORS bootstrap settings. Compose forwards an
// unset host value as the empty string (#888), so empty behaves like unset:
// no allowed origins (deny all cross-origin requests) and no wildcard.
func corsConfigFromEnv() middleware.CORSConfig {
	cfg := middleware.CORSConfig{
		AllowUnsafeWildcard: os.Getenv("NRCC_CORS_UNSAFE_WILDCARD") == "true",
	}
	for _, o := range strings.Split(os.Getenv("NRCC_CORS_ORIGINS"), ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, trimmed)
		}
	}
	return cfg
}
