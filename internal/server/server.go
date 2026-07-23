package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/config"
	"github.com/fgjcarlos/nrcc/internal/handler"
	"github.com/fgjcarlos/nrcc/internal/metrics"
	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/go-chi/chi/v5"
)

// Server represents the HTTP server configuration
type Server struct {
	router           *chi.Mux
	authSvc          *service.AuthService
	processManager   *service.ProcessManager
	hostSvc          *service.HostService
	envSvc           *service.EnvService
	updateSvc        *service.UpdateService
	backupSvc        *service.BackupService
	librarySvc       *service.LibraryService
	envHandler       *handler.EnvHandler
	dockerHandler    *handler.DockerHandler
	systemHandler    *handler.SystemHandler
	metricsCollector *metrics.MetricsCollector
	metricsBuffer    *service.MetricsBuffer
	metricsSampler   *service.MetricsSampler
	flowVersionSvc   *service.FlowVersionService
	ctx              context.Context
	cancel           context.CancelFunc
	shutdownCh       chan struct{}
}

// NewServer creates and configures a new server
func NewServer(authSvc *service.AuthService) *Server {
	return NewServerWithConfig(authSvc, "./data", middleware.CORSConfig{})
}

// NewServerWithConfig creates and configures a new server with config directory
func NewServerWithConfig(authSvc *service.AuthService, dataDir string, corsCfg middleware.CORSConfig) *Server {
	r := chi.NewRouter()

	// Global middleware — Recoverer MUST be first so it wraps every downstream
	// middleware and handler. A panic inside SecurityHeaders, CORS, or Logger
	// would otherwise escape and drop the connection.
	r.Use(middleware.Recoverer)
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
	systemHandler.SetEdgeMode(config.EdgeMode())
	bootstrapHandler := handler.NewBootstrapHandler(hostSvc)

	// MFA service + handler. Wires the auth flow so enrolled users
	// get the second-factor challenge at login.
	mfaSvc := service.NewMfaService(dataDir, authSvc)
	authHandler.SetMfaService(mfaSvc)
	mfaHandler := handler.NewMfaHandler(mfaSvc, authSvc)

	// Initialize MetricsBuffer (120-entry ring buffer) and sampler (30s interval)
	metricsBuffer := service.NewMetricsBuffer(120)
	metricsSampler := service.NewMetricsSampler(metricsBuffer, 30*time.Second)
	systemHandler.SetMetricsBuffer(metricsBuffer)

	// Phase 6 handlers
	backupSvc := service.NewBackupService(dataDir)
	backupHandler := handler.NewBackupHandler(backupSvc)
	if provider, perr := service.NewResticProviderFromEnv(); perr != nil {
		log.Printf("backup: restic provider misconfigured: %v", perr)
	} else if provider != nil {
		backupSvc.SetBackupProvider(provider)
		log.Printf("backup: restic provider configured (%s)", provider.Repo)
	}
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
	// Wire the backup engine so pre-update backups are real archives, not placeholders.
	updateSvc.SetBackupCreator(backupSvc)
	updateHandler := handler.NewUpdateHandler(updateSvc)
	filesHandler := handler.NewFilesHandler(dataDir)
	dockerHandler := handler.NewDockerHandler()
	// DockerService powers the native-host container endpoints
	// (status, info, restart, stop) when nrcc itself runs natively.
	// The handler is the only consumer; the service is intentionally
	// stateless so a single instance is safe to share.
	dockerSvc := service.NewDockerService()
	dockerHandler.SetDockerService(dockerSvc)
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
	flowHandler.SetAuditService(auditSvc)
	authHandler.SetRateLimiter(middleware.NewRateLimiter(dataDir))
	mfaHandler.SetAuditService(auditSvc)
	// MFA verify shares the auth surface's rate limiter instance so
	// the per-IP and per-user buckets used by /api/auth/login also
	// cover /api/auth/mfa/verify. Constructed once and shared.
	mfaHandler.SetRateLimiter(middleware.NewRateLimiter(dataDir))

	// Initialize metrics collector and wire into handlers
	metricsCollector := metrics.NewCollector()
	authHandler.SetLoginMetrics(metricsCollector)
	backupHandler.SetBackupMetrics(metricsCollector)
	libraryHandler.SetLibraryMetrics(metricsCollector)
	updateHandler.SetUpdateMetrics(metricsCollector)

	// Public routes (no auth required)
	// GetHealth replaces the old inline closure; it returns status + uptime +
	// restartCount (the durable cumulative counter, not the backoff one).
	r.Get("/api/health", systemHandler.GetHealth)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/metrics", metricsCollector.Handler().ServeHTTP)

	// Auth routes (public and protected mixed)
	r.Route("/api/auth", func(r chi.Router) {
		// Public auth endpoints
		r.Get("/status", authHandler.GetStatus)
		r.Post("/setup", authHandler.Setup)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/mfa/verify", mfaHandler.Verify)

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
			r.Post("/mfa/enroll", mfaHandler.Enroll)
			r.Post("/mfa/enroll/confirm", mfaHandler.EnrollConfirm)
			r.Post("/mfa/disable", mfaHandler.Disable)
			r.Get("/mfa/status", mfaHandler.Status)
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
		r.Get("/api/system/history", systemHandler.GetSystemHistory)
		r.Get("/api/runtime/history", systemHandler.GetRuntimeHistory)

		// Backup routes — reads are open to any authenticated user; all
		// state-mutating operations require the admin role.
		r.Route("/api/backups", func(r chi.Router) {
			r.Get("/", backupHandler.GetBackups)
			r.With(middleware.RequireAdmin).Post("/", backupHandler.PostBackup)
			r.Get("/status", backupHandler.GetBackupStatus)
			r.Get("/observability", backupHandler.GetBackupObservability)
			r.Get("/storage", backupHandler.GetBackupStorage)
			r.Get("/config", backupHandler.GetBackupConfig)
			r.With(middleware.RequireAdmin).Post("/config", backupHandler.PostBackupConfig)
			r.Get("/provider", backupHandler.GetBackupProvider)
			r.Get("/provider/snapshots", backupHandler.ListProviderSnapshots)
			r.With(middleware.RequireAdmin).Post("/provider/restore", backupHandler.RestoreProviderSnapshot)
			r.Get("/{id}", backupHandler.GetBackupDetail)
			r.With(middleware.RequireAdmin).Delete("/{id}", backupHandler.DeleteBackup)
			r.Get("/{id}/download", backupHandler.DownloadBackup)
			r.With(middleware.RequireAdmin).Post("/{id}/restore", backupHandler.RestoreBackup)
		})

		// Scheduler routes
		r.Route("/api/scheduler", func(r chi.Router) {
			r.With(middleware.RequireAdmin).Post("/config", backupHandler.PostSchedulerConfig)
			r.Get("/history", backupHandler.GetSchedulerHistory)
		})

		// Storage routes
		r.Route("/api/storage", func(r chi.Router) {
			r.With(middleware.RequireAdmin).Patch("/retention", backupHandler.PatchStorageRetention)
		})

		// Environment variable routes
		r.Route("/api/env", func(r chi.Router) {
			r.Get("/", envHandler.GetEnv)
			r.With(middleware.RequireAdmin).Post("/", envHandler.PostEnv)
			r.With(middleware.RequireAdmin).Post("/bulk", envHandler.BulkEnv)
			r.With(middleware.RequireAdmin).Delete("/{key}", envHandler.DeleteEnv)
			r.Get("/dotenv", envHandler.GetDotenv)                               // TAREA 2c: Read .env file
			r.With(middleware.RequireAdmin).Put("/dotenv", envHandler.PutDotenv) // TAREA 2c: Write .env file
		})

		// Flow routes
		r.Route("/api/flows", func(r chi.Router) {
			r.Get("/", flowHandler.GetFlows)
			r.Get("/export", flowHandler.ExportFlows)
			r.Get("/versions", flowHandler.GetVersions)
			r.With(middleware.RequireAdmin).Post("/versions", flowHandler.PostSnapshot)
			r.Get("/versions/{from}/diff/{to}", flowHandler.GetVersionDiff)
			r.With(middleware.RequireAdmin).Post("/versions/{id}/revert", flowHandler.PostRevert)
			r.Get("/{id}", flowHandler.GetFlow)
		})

		// Library routes
		r.Route("/api/libraries", func(r chi.Router) {
			r.Get("/", libraryHandler.GetLibraries)
			r.With(middleware.RequireAdmin).Post("/install", libraryHandler.PostInstall)
			r.Post("/search", libraryHandler.PostSearch)
			r.With(middleware.RequireAdmin).Delete("/{name}", libraryHandler.DeleteLibrary)
			r.Get("/{name}/check", libraryHandler.GetLibraryCheck)
		})

		// Update routes
		r.Route("/api/updates", func(r chi.Router) {
			r.Get("/status", updateHandler.GetStatus)
			r.Get("/check", updateHandler.GetCheck)
			r.Get("/state", updateHandler.GetState)
			r.With(middleware.RequireAdmin).Post("/apply", updateHandler.PostApply)
			r.Get("/history", updateHandler.GetHistory)
		})

		// Files routes
		r.Route("/api/files", func(r chi.Router) {
			r.Get("/", filesHandler.GetList)
			r.With(middleware.RequireAdmin).Post("/upload", filesHandler.PostUpload)
			r.Get("/{name}/download", filesHandler.DownloadFile)
			r.With(middleware.RequireAdmin).Delete("/{name}", filesHandler.DeleteFile)
		})

		// Docker routes — only the read-only /status endpoint survives
		// after #477; the dashboard's status card consumes it. Restart,
		// stop and engine-info endpoints were structurally meaningless
		// under the docker-first model (nrcc would kill its own host).
		r.Route("/api/docker", func(r chi.Router) {
			r.Get("/status", dockerHandler.GetStatus)
		})

		// AI routes
		r.Route("/api/ai", func(r chi.Router) {
			r.Post("/analyze/flow", aiHandler.PostAnalyzeFlow)
		})
	})

	server := &Server{
		router:           r,
		authSvc:          authSvc,
		hostSvc:          hostSvc,
		envSvc:           envSvc,
		updateSvc:        updateSvc,
		backupSvc:        backupSvc,
		librarySvc:       librarySvc,
		envHandler:       envHandler,
		dockerHandler:    dockerHandler,
		systemHandler:    systemHandler,
		metricsCollector: metricsCollector,
		metricsBuffer:    metricsBuffer,
		metricsSampler:   metricsSampler,
		flowVersionSvc:   flowVersionSvc,
	}

	// Create a cancellable context for the server lifecycle
	server.ctx, server.cancel = context.WithCancel(context.Background())

	// Initialize shutdown channel (buffered to prevent goroutine leak)
	server.shutdownCh = make(chan struct{}, 1)

	// Start the backup scheduler using persisted config.
	backupSvc.Start(server.ctx)

	// Start the update service polling goroutine
	server.updateSvc.Start(server.ctx)

	// Start the metrics sampler goroutine (samples CPU/mem/disk every 30s)
	go server.metricsSampler.Start(server.ctx)

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
	if s.flowVersionSvc != nil {
		s.flowVersionSvc.Stop()
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
	// Wire process manager into metrics collector for runtime status gauges
	if s.metricsCollector != nil {
		s.metricsCollector.SetProcessManager(pm)
	}
	// Wire process manager into system handler for runtime history endpoint
	if s.systemHandler != nil {
		s.systemHandler.SetProcessManager(pm)
	}
	// Wire the lifecycle hooks into the backup service so a restore can
	// quiesce Node-RED and restart it after files are swapped into dataDir.
	// Both stops are best-effort; failure is recorded in the backup event
	// stream but does not roll back a successful restore.
	if s.backupSvc != nil {
		stop := pm.Stop
		start := pm.Start
		s.backupSvc.SetRestoreHooks(stop, func() error {
			if err := start(); err != nil {
				return err
			}
			return nil
		})
	}
	// Wire the library service so npm install/uninstall triggers a Node-RED
	// restart and the editor picks up new nodes without a container
	// recreation. The restart is best-effort: a failing hook must not turn
	// into an HTTP 500 on the install endpoint.
	if s.librarySvc != nil {
		start := pm.Start
		s.librarySvc.SetNodeRedRestart(func() error {
			if err := pm.Stop(); err != nil {
				return err
			}
			return start()
		})
	}
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
