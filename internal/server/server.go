package server

import (
	"context"
	"net/http"
	"os"

	"github.com/composedof2/nrcc/internal/audit"
	"github.com/composedof2/nrcc/internal/handler"
	"github.com/composedof2/nrcc/internal/middleware"
	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

// Server represents the HTTP server configuration
type Server struct {
	router         *chi.Mux
	authSvc        *service.AuthService
	processManager *service.ProcessManager
	logBuffer      *service.LogBuffer
	hostSvc        *service.HostService
	envSvc         *service.EnvService
	updateSvc      *service.UpdateService
	envHandler     *handler.EnvHandler
	dockerHandler  *handler.DockerHandler
	ctx            context.Context
	cancel         context.CancelFunc
	shutdownCh     chan struct{}
}

// NewServer creates and configures a new server
func NewServer(authSvc *service.AuthService) *Server {
	return NewServerWithConfig(authSvc, "./data", middleware.CORSConfig{})
}

// NewServerWithConfig creates and configures a new server with config directory
func NewServerWithConfig(authSvc *service.AuthService, dataDir string, corsCfg middleware.CORSConfig) *Server {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS(corsCfg))
	r.Use(middleware.Logger)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authSvc)
	hostSvc := service.NewHostService(dataDir)
	configSvc := service.NewConfigServiceWithHost(dataDir, hostSvc)
	configHandler := handler.NewConfigHandler(configSvc)
	settingsHandler := handler.NewSettingsHandler(configSvc)
	systemHandler := handler.NewSystemHandler()
	bootstrapHandler := handler.NewBootstrapHandler(hostSvc)

	// Phase 6 handlers
	backupSvc := service.NewBackupService(dataDir)
	backupHandler := handler.NewBackupHandler(backupSvc)
	envSvc := service.NewEnvService(configSvc, os.Getenv("NRCC_ENCRYPTION_KEY"))
	envHandler := handler.NewEnvHandler(envSvc, dataDir) // TAREA 2c: Pass dataDir
	flowSvc := service.NewFlowService(dataDir)
	flowVersionSvc := service.NewFlowVersionService(dataDir)
	flowVersionSvc.StartPolling()
	flowHandler := handler.NewFlowHandler(flowSvc)
	flowHandler.SetVersionService(flowVersionSvc)
	librarySvc := service.NewLibraryService(dataDir)
	libraryHandler := handler.NewLibraryHandler(librarySvc)
	updateSvc := service.NewUpdateService(dataDir)
	updateHandler := handler.NewUpdateHandler(updateSvc)
	filesHandler := handler.NewFilesHandler(dataDir)
	dockerHandler := handler.NewDockerHandler()
	aiHandler := handler.NewAIHandler()

	// Initialize audit service
	auditSvc, _ := audit.NewService(dataDir)
	authHandler.SetAuditService(auditSvc)
	configHandler.SetAuditService(auditSvc)
	settingsHandler.SetAuditService(auditSvc)
	backupHandler.SetAuditService(auditSvc)
	envHandler.SetAuditService(auditSvc)
	updateHandler.SetAuditService(auditSvc)
	filesHandler.SetAuditService(auditSvc)
	dockerHandler.SetAuditService(auditSvc)
	flowHandler.SetAuditService(auditSvc)
	authHandler.SetRateLimiter(middleware.NewRateLimiter(dataDir))

	// Public routes (no auth required)
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		model.RespondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "ok",
		})
	})
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Auth routes (public and protected mixed)
	r.Route("/api/auth", func(r chi.Router) {
		// Public auth endpoints
		r.Get("/status", authHandler.GetStatus)
		r.Post("/setup", authHandler.Setup)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)

		// Protected auth endpoints
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authSvc))
			r.Get("/me", authHandler.GetMe)
			r.Post("/logout", authHandler.Logout)
			r.Get("/users", authHandler.GetUsers)
			r.Post("/users", authHandler.CreateUser)
			r.Delete("/users/{id}", authHandler.DeleteUser)
			r.Patch("/users/{id}", authHandler.UpdateUser)
			r.Patch("/users/{id}/password", authHandler.ChangePassword)
		})
	})

	// Protected routes (auth middleware applied)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(authSvc))

		// Config routes
		r.Route("/api/config", func(r chi.Router) {
			r.Get("/", configHandler.GetConfig)
			r.Post("/", configHandler.SaveConfig)
			r.Get("/default", configHandler.GetDefaultConfig)
			r.Post("/validate", configHandler.ValidateConfig)
		})

		r.Route("/api/settings", func(r chi.Router) {
			r.Get("/raw", settingsHandler.GetRaw)
			r.Post("/raw", settingsHandler.SaveRaw)
		})

		r.Route("/api/bootstrap", func(r chi.Router) {
			r.Get("/status", bootstrapHandler.GetStatus)
		})

		// System routes
		r.Get("/api/system/info", systemHandler.GetSystemInfo)

		// Backup routes
		r.Route("/api/backups", func(r chi.Router) {
			r.Get("/", backupHandler.GetBackups)
			r.Post("/", backupHandler.PostBackup)
			r.Get("/status", backupHandler.GetBackupStatus)
			r.Get("/observability", backupHandler.GetBackupObservability)
			r.Get("/storage", backupHandler.GetBackupStorage)
			r.Get("/config", backupHandler.GetBackupConfig)
			r.Post("/config", backupHandler.PostBackupConfig)
			r.Get("/{id}", backupHandler.GetBackupDetail)
			r.Delete("/{id}", backupHandler.DeleteBackup)
			r.Get("/{id}/download", backupHandler.DownloadBackup)
			r.Post("/{id}/restore", backupHandler.RestoreBackup)
		})

		// Scheduler routes
		r.Route("/api/scheduler", func(r chi.Router) {
			r.Post("/config", backupHandler.PostSchedulerConfig)
			r.Get("/history", backupHandler.GetSchedulerHistory)
		})

		// Storage routes
		r.Route("/api/storage", func(r chi.Router) {
			r.Patch("/retention", backupHandler.PatchStorageRetention)
		})

		// Environment variable routes
		r.Route("/api/env", func(r chi.Router) {
			r.Get("/", envHandler.GetEnv)
			r.Post("/", envHandler.PostEnv)
			r.Delete("/{key}", envHandler.DeleteEnv)
			r.Get("/dotenv", envHandler.GetDotenv) // TAREA 2c: Read .env file
			r.Put("/dotenv", envHandler.PutDotenv) // TAREA 2c: Write .env file
		})

		// Flow routes
		r.Route("/api/flows", func(r chi.Router) {
			r.Get("/", flowHandler.GetFlows)
			r.Get("/export", flowHandler.ExportFlows)
			r.Post("/analyze", flowHandler.AnalyzeFlows)
			r.Get("/versions", flowHandler.GetVersions)
			r.Post("/versions", flowHandler.PostSnapshot)
			r.Get("/versions/{from}/diff/{to}", flowHandler.GetVersionDiff)
			r.Post("/versions/{id}/revert", flowHandler.PostRevert)
			r.Get("/{id}", flowHandler.GetFlow)
		})

		// Library routes
		r.Route("/api/libraries", func(r chi.Router) {
			r.Get("/", libraryHandler.GetLibraries)
			r.Post("/install", libraryHandler.PostInstall)
			r.Post("/search", libraryHandler.PostSearch)
			r.Delete("/{name}", libraryHandler.DeleteLibrary)
			r.Get("/{name}/check", libraryHandler.GetLibraryCheck)
		})

		// Update routes
		r.Route("/api/updates", func(r chi.Router) {
			r.Get("/status", updateHandler.GetStatus)
			r.Get("/check", updateHandler.GetCheck)
			r.Get("/state", updateHandler.GetState)
			r.Post("/apply", updateHandler.PostApply)
			r.Get("/history", updateHandler.GetHistory)
		})

		// Files routes
		r.Route("/api/files", func(r chi.Router) {
			r.Get("/", filesHandler.GetList)
			r.Post("/upload", filesHandler.PostUpload)
			r.Delete("/{name}", filesHandler.DeleteFile)
		})

		// Docker routes
		r.Route("/api/docker", func(r chi.Router) {
			r.Get("/status", dockerHandler.GetStatus)
			r.Get("/info", dockerHandler.GetInfo)
			r.Post("/restart", dockerHandler.PostRestart)
			r.Post("/stop", dockerHandler.PostStop)
		})

		// AI routes
		r.Route("/api/ai", func(r chi.Router) {
			r.Post("/analyze/flow", aiHandler.PostAnalyzeFlow)
			r.Post("/analyze/patterns", aiHandler.PostAnalyzePatterns)
			r.Get("/patterns/{id}/readme", aiHandler.GetPatternReadme)
			r.Get("/patterns/{id}/download", aiHandler.DownloadPattern)
		})
	})

	server := &Server{
		router:         r,
		authSvc:        authSvc,
		hostSvc:        hostSvc,
		envSvc:         envSvc,
		updateSvc:      updateSvc,
		envHandler:     envHandler,
		dockerHandler:  dockerHandler,
	}

	// Create a cancellable context for the server lifecycle
	server.ctx, server.cancel = context.WithCancel(context.Background())

	// Initialize shutdown channel (buffered to prevent goroutine leak)
	server.shutdownCh = make(chan struct{}, 1)

	// Start the backup scheduler using persisted config.
	backupSvc.Start(server.ctx)

	// Start the update service polling goroutine
	server.updateSvc.Start(server.ctx)

	// SPA fallback (must be last)
	r.Handle("/*", SPAHandler(embedFS))

	return server
}

// Shutdown gracefully shuts down the server and its services
func (s *Server) Shutdown() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.updateSvc != nil {
		s.updateSvc.Stop()
	}
}

// GetShutdownChannel returns the shutdown channel for handler-triggered shutdown signals
func (s *Server) GetShutdownChannel() chan struct{} {
	return s.shutdownCh
}

// SetProcessManager sets the ProcessManager for runtime routes
func (s *Server) SetProcessManager(pm *service.ProcessManager) {
	s.processManager = pm
	// Wire env vars into the process so they're injected on every node-red start
	pm.SetEnvService(s.envSvc)
	// Wire process manager into env handler so it restarts node-red on env changes
	s.envHandler.SetProcessManager(pm)
	// Wire process manager into docker handler so container restart stops node-red first
	s.dockerHandler.SetProcessManager(pm)
	// Wire shutdown channel into docker handler for graceful shutdown signaling
	s.dockerHandler.SetShutdownChannel(s.shutdownCh)
}

// SetLogBuffer sets the LogBuffer for log streaming routes
func (s *Server) SetLogBuffer(lb *service.LogBuffer) {
	s.logBuffer = lb
	// Re-wire routes with log buffer
	s.wireLogRoutes()
}

// wireLogRoutes adds log routes to the existing router
func (s *Server) wireLogRoutes() {
	if s.logBuffer == nil {
		return
	}
	logHandler := handler.NewLogHandler(s.logBuffer)
	s.router.Route("/api/logs", func(r chi.Router) {
		r.Use(middleware.Auth(s.authSvc))
		r.Get("/", logHandler.GetLogs)
		r.Get("/stream", logHandler.StreamLogs)
		r.Delete("/", logHandler.DeleteLogs)
	})
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
