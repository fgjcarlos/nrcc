package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/composedof2/nrcc/cmd"
	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/server"
	"github.com/composedof2/nrcc/internal/service"
	"github.com/composedof2/nrcc/internal/store"
	"github.com/composedof2/nrcc/internal/ui"
)

var Version string = "dev"

func main() {
	// Initialize UI (pterm configuration)
	ui.Init()

	// If arguments provided, route to CLI
	if len(os.Args) > 1 {
		if err := cmd.Execute(); err != nil {
			os.Exit(1)
		}
		return
	}

	// No arguments: run server
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

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		ui.Warn("JWT_SECRET not set, using insecure default. Set JWT_SECRET in production.")
		jwtSecret = "cc-secret-change-in-production"
	}

	// Print startup banner
	ui.Banner(Version, port, dataDir)

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize JSON store for users
	usersPath := filepath.Join(dataDir, "cc-users.json")
	usersStore := store.NewJSONStore[model.CCUsers](usersPath)

	// Initialize auth service
	authSvc := service.NewAuthService(jwtSecret, usersStore)

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
	pm.SetEnvService(service.NewEnvService(configSvc))

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

	// Set embedded filesystem in server package
	server.SetEmbedFS(frontendFS)

	// Create and configure server
	srv := server.NewServerWithConfig(authSvc, dataDir)
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
