package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/config"
	"github.com/fgjcarlos/nrcc/internal/handler"
	"github.com/fgjcarlos/nrcc/internal/metrics"
	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	setupstate "github.com/fgjcarlos/nrcc/internal/setup"
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
	configHandler    *handler.ConfigHandler
	settingsHandler  *handler.SettingsHandler
	flowHandler      *handler.FlowHandler
	systemHandler    *handler.SystemHandler
	metricsCollector *metrics.MetricsCollector
	metricsBuffer    *service.MetricsBuffer
	metricsSampler   *service.MetricsSampler
	flowVersionSvc   *service.FlowVersionService
	httpLogger       *slog.Logger
	ctx              context.Context
	cancel           context.CancelFunc
	shutdownCh       chan struct{}
}

func initAuditService(dataDir string, reportf func(string, ...any)) *audit.Service {
	svc, err := audit.NewService(dataDir)
	if err != nil {
		reportf("audit initialization failed: %v", err)
		return nil
	}
	return svc
}

// metricsArePublic reports whether the Prometheus /metrics endpoint should
// be served unauthenticated. Default is false (#671) — the endpoint is
// fingerprintable and the login-failure counter is a brute-force oracle.
// Set NRCC_METRICS_PUBLIC=true to restore the previous open behaviour, for
// example when the metrics port is bound to a private network.
func metricsArePublic() bool {
	return strings.EqualFold(os.Getenv("NRCC_METRICS_PUBLIC"), "true")
}

// NewServer creates and configures a new server
func NewServer(authSvc *service.AuthService) *Server {
	return NewServerWithConfig(authSvc, Config{DataDir: "./data", HTTPLogger: NewDefaultHTTPLogger()})
}

// NewServerWithConfig creates and configures a new server.
func NewServerWithConfig(authSvc *service.AuthService, cfg Config) *Server {
	if cfg.HTTPLogger == nil {
		panic("server: nil HTTP logger")
	}
	dataDir := cfg.DataDir
	r := chi.NewRouter()

	// Global middleware — Recoverer MUST be first so it wraps every downstream
	// middleware and handler. A panic inside SecurityHeaders, CORS, or Logger
	// would otherwise escape and drop the connection.
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(cfg.HTTPLogger))
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS(cfg.CORS))
	r.Use(middleware.BodyLimitMiddleware(middleware.DefaultBodyLimitConfig()))

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authSvc)
	setupTokenPath := filepath.Join(dataDir, setupstate.SetupTokenFileName)
	users, _ := authSvc.GetAllUsers()
	_, err := setupstate.EnsureTokenFile(setupTokenPath, len(users) > 0)
	if err != nil {
		log.Printf("auth setup token unavailable: %v", err)
	}
	authHandler.SetSetupTokenPath(setupTokenPath)
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
	encKey := os.Getenv("NRCC_ENCRYPTION_KEY")
	if err := service.ValidateSecret("NRCC_ENCRYPTION_KEY", encKey); err != nil {
		log.Fatalf("Encryption key error: %v", err)
	}
	// #664: when the operator boots the binary without an encryption key
	// but the persisted config already contains env vars flagged
	// Encrypted: true, those entries are at rest in cleartext. Surface
	// the count so the operator can either set a key and restart, or
	// rotate the key after wiping the existing entries. The new
	// fail-closed write path means new writes will be rejected, but
	// the pre-existing plaintext stays in config.json until removed.
	if encKey == "" {
		if cfg, cfgErr := configSvc.Get(); cfgErr == nil {
			count := countEncryptedEntries(cfg.EnvVars)
			if count > 0 {
				log.Printf("WARN: NRCC_ENCRYPTION_KEY is empty but %d encrypted entry(ies) exist in config.json — they are stored as plaintext. Set NRCC_ENCRYPTION_KEY, restart, and consider rotating the key after wiping the affected entries (#664)", count)
			}
		} else {
			log.Printf("WARN: could not load config to scan for plaintext encrypted entries: %v", cfgErr)
		}
	}
	envSvc := service.NewEnvService(configSvc, encKey)
	envHandler := handler.NewEnvHandler(envSvc, dataDir) // TAREA 2c: Pass dataDir
	// #676 item 2: wire the security-posture endpoint dependencies.
	systemHandler.SetEnvService(envSvc)
	systemHandler.SetAuthService(authSvc)
	systemHandler.SetMfaService(mfaSvc)
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
	auditSvc := initAuditService(dataDir, log.Printf)
	authHandler.SetAuditService(auditSvc)
	configHandler.SetAuditService(auditSvc)
	settingsHandler.SetAuditService(auditSvc)
	backupHandler.SetAuditService(auditSvc)
	envHandler.SetAuditService(auditSvc)
	updateHandler.SetAuditService(auditSvc)
	filesHandler.SetAuditService(auditSvc)
	flowHandler.SetAuditService(auditSvc)
	// One RateLimiter for the whole auth surface. Two instances would keep
	// independent in-memory bucket maps while persisting to the same
	// ratelimit.json, so each could overwrite the other's buckets from a
	// stale map and drop lockouts. Closes #615.
	rateLimiter := middleware.NewRateLimiter(dataDir)
	authHandler.SetRateLimiter(rateLimiter)
	mfaHandler.SetAuditService(auditSvc)
	// MFA verify shares the auth surface's rate limiter instance so
	// the per-IP and per-user buckets used by /api/auth/login also
	// cover /api/auth/mfa/verify. Constructed once and shared.
	mfaHandler.SetRateLimiter(rateLimiter)

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
	// Public image assets referenced by Node-RED's editorTheme. The handler
	// verifies content bytes and refuses to serve non-images.
	r.Get("/uploads/{name}", filesHandler.ServeImage)

	// #671: /metrics is gated by default — it leaks login-failure counters
	// (a brute-force oracle) and Go runtime fingerprinting data. Operators
	// who run Prometheus on a private network can opt out with
	// NRCC_METRICS_PUBLIC=true. The route is therefore conditionally
	// registered in the public group or the auth group below.
	if metricsArePublic() {
		r.Get("/metrics", metricsCollector.Handler().ServeHTTP)
	}

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
			r.With(middleware.RequireAdmin).Get("/users", authHandler.GetUsers)
			r.With(middleware.RequireAdmin).Post("/users", authHandler.CreateUser)
			r.With(middleware.RequireAdmin).Delete("/users/{id}", authHandler.DeleteUser)
			r.With(middleware.RequireAdmin).Patch("/users/{id}", authHandler.UpdateUser)
			// Self-or-admin: a user may change their own password; an admin
			// may change anyone's. The target is the path {id}; the
			// middleware extractor below pulls it from the route context.
			r.With(middleware.RequireSelfOrAdmin(changePasswordTarget)).Patch("/users/{id}/password", authHandler.ChangePassword)
			r.Post("/mfa/enroll", mfaHandler.Enroll)
			r.Post("/mfa/enroll/confirm", mfaHandler.EnrollConfirm)
			// Self-or-admin: target userId lives in the request body, so the
			// extractor peeks and restores r.Body. See mfaDisableTarget.
			r.With(middleware.RequireSelfOrAdmin(mfaDisableTarget)).Post("/mfa/disable", mfaHandler.Disable)
			r.Get("/mfa/status", mfaHandler.Status)
		})
	})

	// Protected routes (auth middleware applied)
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(authSvc))
		// Baseline per-IP throttle for every authenticated route. A valid JWT
		// was previously enough to hammer any endpoint unthrottled (#585
		// HIGH-002). Generous enough for normal dashboard polling; expensive
		// endpoints below get their own tighter caps.
		r.Use(middleware.RateLimitIP(300, time.Minute))

		// #671: when NRCC_METRICS_PUBLIC is not set, /metrics requires
		// the same JWT as every other authenticated route. A Prometheus
		// scrape config can supply the bearer token directly.
		if !metricsArePublic() {
			r.Get("/metrics", metricsCollector.Handler().ServeHTTP)
		}

		// Config routes
		r.Route("/api/config", func(r chi.Router) {
			r.Get("/", configHandler.GetConfig)
			r.With(middleware.RequireAdmin).Post("/", configHandler.SaveConfig)
			r.Get("/default", configHandler.GetDefaultConfig)
			r.Post("/validate", configHandler.ValidateConfig)
		})

		r.Route("/api/settings", func(r chi.Router) {
			// MEDIUM-016: settings.js carries adminAuth hashes and secrets.
			// Both reads and writes are admin-only on the router.
			r.With(middleware.RequireAdmin).Get("/raw", settingsHandler.GetRaw)
			r.With(middleware.RequireAdmin).Post("/raw", settingsHandler.SaveRaw)
		})

		r.Route("/api/bootstrap", func(r chi.Router) {
			r.Get("/status", bootstrapHandler.GetStatus)
		})

		// System routes
		r.Get("/api/system/info", systemHandler.GetSystemInfo)
		r.Get("/api/system/history", systemHandler.GetSystemHistory)
		// #715: dashboard restart/start/stop buttons were hitting the SPA
		// fallback (200 HTML) instead of a real handler. Wire them to the
		// SystemHandler's runtime controls (admin-only — these mutate the
		// managed Node-RED process).
		r.With(middleware.RequireAdmin).Post("/api/runtime/restart", systemHandler.RestartNodeRed)
		r.With(middleware.RequireAdmin).Post("/api/runtime/start", systemHandler.StartNodeRed)
		r.With(middleware.RequireAdmin).Post("/api/runtime/stop", systemHandler.StopNodeRed)
		r.Get("/api/runtime/history", systemHandler.GetRuntimeHistory)
		// #676 item 2: backs the SecurityPostureCard. Returns the four
		// boolean/count chips that surface the silent-degradation failure
		// mode in issue #04 (encrypted env vars written in clear when
		// NRCC_ENCRYPTION_KEY is missing).
		r.With(middleware.RequireAdmin).Get("/api/system/security-posture", systemHandler.GetSecurityPosture)

		// Backup routes — reads are open to any authenticated user; all
		// state-mutating operations require the admin role.
		r.Route("/api/backups", func(r chi.Router) {
			r.Get("/", backupHandler.GetBackups)
			r.With(middleware.RateLimitIP(10, time.Hour), middleware.RequireAdmin).Post("/", backupHandler.PostBackup)
			r.Get("/status", backupHandler.GetBackupStatus)
			r.Get("/observability", backupHandler.GetBackupObservability)
			r.Get("/storage", backupHandler.GetBackupStorage)
			r.Get("/config", backupHandler.GetBackupConfig)
			r.With(middleware.RequireAdmin).Post("/config", backupHandler.PostBackupConfig)
			r.Get("/provider", backupHandler.GetBackupProvider)
			// #675: remote-snapshot list exposes provider name, snapshot ids,
			// timestamps, and the remote repository layout. The neighbouring
			// GetBackupProvider already treats this as fingerprintable, so the
			// same protection applies here. Admin-only.
			r.With(middleware.RequireAdmin).Get("/provider/snapshots", backupHandler.ListProviderSnapshots)
			r.With(middleware.RateLimitIP(10, time.Hour), middleware.RequireAdmin).Post("/provider/restore", backupHandler.RestoreProviderSnapshot)
			r.Get("/{id}", backupHandler.GetBackupDetail)
			r.With(middleware.RequireAdmin).Delete("/{id}", backupHandler.DeleteBackup)
			// #674: the archive contains cc-users.json (bcrypt hashes) plus
			// flows_cred.json. A viewer must not be able to download it.
			r.With(middleware.RequireAdmin).Get("/{id}/download", backupHandler.DownloadBackup)
			r.With(middleware.RateLimitIP(10, time.Hour), middleware.RequireAdmin).Post("/{id}/restore", backupHandler.RestoreBackup)
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
			r.With(middleware.RequireAdmin).Post("/import-from-node-red", envHandler.ImportFromNodeRedEnv)
			r.With(middleware.RequireAdmin).Delete("/{key}", envHandler.DeleteEnv)
			// #673: GET /api/env/dotenv returns the raw .env file unfiltered,
			// defeating the masking that GET /api/env applies. The endpoint
			// exists to power DotenvEditor.tsx; the same caller already
			// requires admin for the structured env surface, so require it
			// here too. Closed: #673.
			r.With(middleware.RequireAdmin).Get("/dotenv", envHandler.GetDotenv) // TAREA 2c: Read .env file
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
			r.Get("/{id}/metrics", flowHandler.GetFlowMetrics)
			r.Get("/{id}", flowHandler.GetFlow)
		})

		// Library routes
		r.Route("/api/libraries", func(r chi.Router) {
			r.Get("/", libraryHandler.GetLibraries)
			// npm install shells out and is slow — tightest cap on the router.
			r.With(middleware.RateLimitIP(5, time.Hour), middleware.RequireAdmin).Post("/install", libraryHandler.PostInstall)
			r.With(middleware.RateLimitIP(60, time.Hour)).Post("/search", libraryHandler.PostSearch)
			r.With(middleware.RequireAdmin).Delete("/{name}", libraryHandler.DeleteLibrary)
			// Proxies the npm registry; unthrottled it is a registry enumeration oracle.
			r.With(middleware.RateLimitIP(60, time.Hour)).Get("/{name}/check", libraryHandler.GetLibraryCheck)
		})

		// Update routes
		r.Route("/api/updates", func(r chi.Router) {
			r.Get("/status", updateHandler.GetStatus)
			r.Get("/check", updateHandler.GetCheck)
			r.Get("/state", updateHandler.GetState)
			r.With(middleware.RateLimitIP(5, time.Hour), middleware.RequireAdmin).Post("/apply", updateHandler.PostApply)
			r.Get("/history", updateHandler.GetHistory)
		})

		// Files routes
		r.Route("/api/files", func(r chi.Router) {
			r.Get("/", filesHandler.GetList)
			r.With(middleware.RateLimitIP(30, time.Hour), middleware.RequireAdmin).Post("/upload", filesHandler.PostUpload)
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
			r.With(middleware.RateLimitIP(20, time.Hour)).Post("/analyze/flow", aiHandler.PostAnalyzeFlow)
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
		configHandler:    configHandler,
		settingsHandler:  settingsHandler,
		flowHandler:      flowHandler,
		systemHandler:    systemHandler,
		metricsCollector: metricsCollector,
		metricsBuffer:    metricsBuffer,
		metricsSampler:   metricsSampler,
		flowVersionSvc:   flowVersionSvc,
		httpLogger:       cfg.HTTPLogger,
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
	// Wire process manager into config handler so a successful SaveConfig
	// auto-restarts Node-RED and the new settings.js is actually picked up
	// by the running process (#715).
	if s.configHandler != nil {
		s.configHandler.SetProcessManager(pm)
	}
	if s.settingsHandler != nil {
		s.settingsHandler.SetProcessManager(pm)
	}
	if s.flowHandler != nil {
		s.flowHandler.SetProcessManager(pm)
	}
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
	// Wire the library service so npm install/uninstall reloads a running
	// Node-RED process. Restart failures propagate to the caller; otherwise
	// the UI would claim the node set is active when only package.json changed.
	if s.librarySvc != nil {
		start := pm.Start
		s.librarySvc.SetNodeRedRestart(func() error {
			if pm.Status().Status != "running" {
				return nil
			}
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

// countEncryptedEntries returns the number of persisted env vars flagged
// Encrypted: true. Used at startup to surface a WARN when the binary
// boots without NRCC_ENCRYPTION_KEY but the config still contains entries
// that would be at rest in cleartext. See #664.
func countEncryptedEntries(envVars []model.EnvVar) int {
	n := 0
	for _, ev := range envVars {
		if ev.Encrypted {
			n++
		}
	}
	return n
}

// changePasswordTarget extracts the target user id from the {id} path
// parameter for PATCH /api/auth/users/{id}/password. Returned alongside
// ok so RequireSelfOrAdmin can reject requests where the path is missing
// the id entirely.
func changePasswordTarget(r *http.Request) (string, bool) {
	id := chi.URLParam(r, "id")
	return id, id != ""
}

// mfaDisableTarget extracts the target user id from the JSON body of
// POST /api/auth/mfa/disable. The body is read, decoded, and restored so
// the handler sees the same bytes; the extractor runs once at middleware
// dispatch and the handler runs once at serve time.
//
// Reading the body twice is cheap: the payload is a small JSON control
// plane message. We refuse to follow up with the service if the JSON is
// unparseable, returning ok=false so the middleware produces a clean 403
// rather than letting a malformed request reach the handler and crash.
func mfaDisableTarget(r *http.Request) (string, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", false
	}
	return req.UserID, true
}
