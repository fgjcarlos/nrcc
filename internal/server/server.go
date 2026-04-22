package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"nrcc/internal/middleware"
	"nrcc/internal/model"
	"nrcc/internal/service"
)

type Config struct {
	Port        string
	Frontend    fs.FS
	Runtime     *service.ProcessManager
	Auth        *service.AuthService
	Config      service.ConfigService
	ManagedEnv  service.ManagedEnvService
	Backups     service.BackupService
	Libraries   service.LibraryService
	Flows       service.FlowService
	Updates     service.UpdateService
	Operations  *service.OperationLock
	Logs        *service.LogService
	Jobs        *service.JobsService
	Doctor      *service.DoctorService
	Support     *service.SupportBundleService
	LocalAccess *service.LocalAccessService
	Assets      *service.AssetService
}

type Server struct {
	httpServer *http.Server
}

type authResponse struct {
	User      *model.UserPublic `json:"user"`
	CSRFToken string            `json:"csrfToken"`
}

// Session cookies stay on same-site requests only. The API already uses an
// explicit CSRF token for authenticated state changes, and Strict avoids
// silently broadening the browser cookie contract across login, register,
// protected POST, and logout flows.
const sessionCookieSameSite = http.SameSiteStrictMode

func New(cfg Config) *Server {
	router := chi.NewRouter()

	// Global middleware: request ID and request logging
	router.Use(middleware.RequestID)
	router.Use(middleware.RequestLogger)

	registerAPIRoutes(router, cfg.Runtime, cfg.Auth, cfg.Config, cfg.ManagedEnv, cfg.Backups, cfg.Libraries, cfg.Flows, cfg.Updates, cfg.Operations, cfg.LocalAccess, cfg.Assets)
	registerDiagnosticsRoutes(router, cfg)
	registerAssetRoutes(router, cfg.Auth, cfg.Assets)
	registerSPARoutes(router, cfg.Frontend)

	return &Server{
		httpServer: &http.Server{
			Addr:              ":" + cfg.Port,
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func registerAPIRoutes(router chi.Router, runtimeManager *service.ProcessManager, authService *service.AuthService, configService service.ConfigService, managedEnvService service.ManagedEnvService, backupService service.BackupService, libraryService service.LibraryService, flowService service.FlowService, updateService service.UpdateService, operationLock *service.OperationLock, localAccessService *service.LocalAccessService, assetService *service.AssetService) {
	if operationLock == nil {
		operationLock = service.NewOperationLock()
	}

	router.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		respondOK(w, map[string]any{
			"status": "ok",
		})
	})

	if authService != nil {
		router.Route("/api/auth", func(r chi.Router) {
			r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
				hasUsers, err := authService.HasUsers()
				if err != nil {
					respondError(w, http.StatusInternalServerError, "AUTH_STATUS_FAILED", err.Error())
					return
				}

				respondOK(w, map[string]any{
					"hasUsers": hasUsers,
				})
			})

			r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
				var payload struct {
					Username string `json:"username"`
					Password string `json:"password"`
				}
				if err := decodeJSON(r, &payload); err != nil {
					respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
					return
				}

				user, token, err := authService.RegisterInitial(payload.Username, payload.Password)
				if err != nil {
					respondError(w, http.StatusBadRequest, "AUTH_REGISTER_FAILED", err.Error())
					return
				}

				writeSessionCookie(w, r, token, authService.SessionTTL())
				respondOK(w, authResponse{
					User:      user,
					CSRFToken: authService.CSRFToken(token),
				})
			})

			r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
				var payload struct {
					Username string `json:"username"`
					Password string `json:"password"`
				}
				if err := decodeJSON(r, &payload); err != nil {
					respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
					return
				}

				user, token, err := authService.Login(payload.Username, payload.Password, requestClientAddress(r))
				if err != nil {
					respondError(w, http.StatusUnauthorized, "AUTH_LOGIN_FAILED", err.Error())
					return
				}

				writeSessionCookie(w, r, token, authService.SessionTTL())
				respondOK(w, authResponse{
					User:      user,
					CSRFToken: authService.CSRFToken(token),
				})
			})

			r.With(requireAuth(authService)).Get("/me", func(w http.ResponseWriter, r *http.Request) {
				claims, ok := middleware.AuthClaimsFromContext(r.Context())
				if !ok {
					respondError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "authentication required")
					return
				}

				user, err := authService.FindPublicUserByID(claims.Sub)
				if err != nil {
					respondError(w, http.StatusInternalServerError, "AUTH_LOOKUP_FAILED", err.Error())
					return
				}
				if user == nil {
					respondError(w, http.StatusUnauthorized, "AUTH_INVALID", "invalid session")
					return
				}

				cookie, err := r.Cookie(service.SessionCookieName)
				if err != nil || cookie.Value == "" {
					respondError(w, http.StatusUnauthorized, "AUTH_INVALID", "invalid session")
					return
				}

				respondOK(w, authResponse{
					User:      user,
					CSRFToken: authService.CSRFToken(cookie.Value),
				})
			})

			r.With(requireAuth(authService), requireCSRF(authService)).Post("/logout", func(w http.ResponseWriter, r *http.Request) {
				cookie, err := r.Cookie(service.SessionCookieName)
				if err == nil && cookie.Value != "" {
					if revokeErr := authService.RevokeToken(cookie.Value); revokeErr != nil {
						respondError(w, http.StatusUnauthorized, "AUTH_INVALID", "invalid session")
						return
					}
				}
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("auth.logout", claims.Username, "logout succeeded")
				}
				clearSessionCookie(w, r)
				respondOK(w, map[string]any{"loggedOut": true})
			})
		})
	}

	router.Group(func(r chi.Router) {
		if authService != nil {
			r.Use(middleware.RequireAuth(authService))
			r.Use(middleware.RequireCSRF(authService))
		}

		if runtimeManager != nil {
			r.Get("/api/runtime/status", func(w http.ResponseWriter, r *http.Request) {
				respondOK(w, runtimeManager.Status())
			})

			r.Get("/api/runtime/logs", func(w http.ResponseWriter, r *http.Request) {
				respondOK(w, map[string]any{
					"lines": runtimeManager.Logs(200),
				})
			})

			r.Post("/api/runtime/restart", func(w http.ResponseWriter, r *http.Request) {
				if err := runtimeManager.Restart(); err != nil {
					respondError(w, http.StatusInternalServerError, "RUNTIME_RESTART_FAILED", err.Error())
					return
				}

				if authService != nil {
					if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
						authService.LogAudit("runtime.restart", claims.Username, "runtime restart requested")
					}
				}

				respondOK(w, runtimeManager.Status())
			})
		}

		r.Get("/api/system/info", func(w http.ResponseWriter, r *http.Request) {
			hostname, _ := os.Hostname()
			localAccess := model.LocalAccessStatus{
				Mode:        "direct",
				URL:         "http://127.0.0.1:3000",
				FallbackURL: "http://127.0.0.1:3000",
				Operational: true,
				Message:     "Direct local access is available.",
			}
			if localAccessService != nil {
				localAccess = localAccessService.Status()
			}

			respondOK(w, model.SystemInfo{
				GOOS:        runtime.GOOS,
				GOARCH:      runtime.GOARCH,
				CPUs:        runtime.NumCPU(),
				Hostname:    hostname,
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
				LocalAccess: localAccess,
			})
		})

		r.Get("/api/operations/status", func(w http.ResponseWriter, r *http.Request) {
			respondOK(w, operationLock.Status())
		})

		// F.1: GET /api/config — Load full config
		r.Get("/api/config", func(w http.ResponseWriter, r *http.Request) {
			cfg, err := configService.LoadFullConfig()
			if err != nil {
				respondError(w, http.StatusInternalServerError, "CONFIG_LOAD_FAILED", err.Error())
				return
			}
			respondOK(w, cfg)
		})

		// F.2: POST /api/config/validate — Validate full config
		r.Post("/api/config/validate", func(w http.ResponseWriter, r *http.Request) {
			var payload model.FullAppConfig
			if err := decodeJSON(r, &payload); err != nil {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
				return
			}
			respondOK(w, configService.ValidateConfig(payload))
		})

		// F.3: POST /api/config/apply — Apply full config
		r.Post("/api/config/apply", func(w http.ResponseWriter, r *http.Request) {
			var payload model.FullAppConfig
			if err := decodeJSON(r, &payload); err != nil {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
				return
			}

			username := "unknown"
			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					username = claims.Username
				}
			}

			result, err := configService.ApplyConfig(payload, username)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "CONFIG_APPLY_FAILED", err.Error())
				return
			}
			if !result.Valid {
				var errMessages []string
				for _, fe := range result.Errors {
					errMessages = append(errMessages, fe.Message)
				}
				respondError(w, http.StatusBadRequest, "CONFIG_INVALID", strings.Join(errMessages, "; "))
				return
			}

			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("config.apply", claims.Username, "full config applied")
				}
			}

			respondOK(w, result)
		})

		// F.4: GET /api/config/schema — Get config JSON schema
		r.Get("/api/config/schema", func(w http.ResponseWriter, r *http.Request) {
			respondOK(w, configService.GetConfigSchema())
		})

		// F.5: GET /api/config/preview — Preview current config as settings.js
		r.Get("/api/config/preview", func(w http.ResponseWriter, r *http.Request) {
			preview, err := configService.PreviewConfig(nil)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "CONFIG_PREVIEW_FAILED", err.Error())
				return
			}
			respondOK(w, map[string]any{
				"settingsJs": preview,
			})
		})

		// F.6: POST /api/config/preview — Preview with provided config
		r.Post("/api/config/preview", func(w http.ResponseWriter, r *http.Request) {
			var payload model.FullAppConfig
			if err := decodeJSON(r, &payload); err != nil {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
				return
			}

			preview, err := configService.PreviewConfig(&payload)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "CONFIG_PREVIEW_FAILED", err.Error())
				return
			}
			respondOK(w, map[string]any{
				"settingsJs": preview,
			})
		})

		// F.7: POST /api/config/backup — Create config backup/snapshot
		r.Post("/api/config/backup", func(w http.ResponseWriter, r *http.Request) {
			var payload struct {
				Label string `json:"label"`
			}
			if err := decodeJSON(r, &payload); err != nil {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
				return
			}

			// Get current config and render settings
			currentCfg, err := configService.LoadFullConfig()
			if err != nil {
				respondError(w, http.StatusInternalServerError, "CONFIG_LOAD_FAILED", err.Error())
				return
			}

			currentJSON, _ := json.Marshal(currentCfg)
			settingsJS := service.RenderSettingsJS(currentCfg)

			username := "unknown"
			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					username = claims.Username
				}
			}

			// Create snapshot
			snapshot, err := configService.CreateSnapshot(payload.Label, "Manual backup by "+username, string(currentJSON), settingsJS)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "BACKUP_CREATE_FAILED", err.Error())
				return
			}

			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("config.backup", claims.Username, "config backup created")
				}
			}

			respondOK(w, snapshot)
		})

		// F.8: GET /api/config/backups — List config snapshots
		r.Get("/api/config/backups", func(w http.ResponseWriter, r *http.Request) {
			snapshots, err := configService.ListSnapshots()
			if err != nil {
				respondError(w, http.StatusInternalServerError, "BACKUPS_LIST_FAILED", err.Error())
				return
			}
			respondOK(w, snapshots)
		})

		// F.9: POST /api/config/backups/:id/restore — Restore from snapshot
		r.Post("/api/config/backups/{id}/restore", func(w http.ResponseWriter, r *http.Request) {
			snapshotID := strings.TrimSpace(chi.URLParam(r, "id"))
			if snapshotID == "" {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "snapshot id is required")
				return
			}

			// Get snapshot
			snapshot, err := configService.GetSnapshot(snapshotID)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "SNAPSHOT_LOOKUP_FAILED", err.Error())
				return
			}
			if snapshot == nil {
				respondError(w, http.StatusNotFound, "SNAPSHOT_NOT_FOUND", "snapshot not found")
				return
			}

			// Create preventive snapshot of current config
			currentCfg, _ := configService.LoadFullConfig()
			currentJSON, _ := json.Marshal(currentCfg)
			currentSettingsJS := service.RenderSettingsJS(currentCfg)

			preventiveSnapshot, err := configService.CreateSnapshot(
				"Pre-restore snapshot",
				"Created before restoring snapshot "+snapshotID,
				string(currentJSON),
				currentSettingsJS,
			)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "PREVENTIVE_SNAPSHOT_FAILED", err.Error())
				return
			}

			// Unmarshal the stored config_json back to FullAppConfig
			var restoredCfg model.FullAppConfig
			if err := json.Unmarshal([]byte(snapshot.ConfigJSON), &restoredCfg); err != nil {
				respondError(w, http.StatusInternalServerError, "CONFIG_UNMARSHAL_FAILED", err.Error())
				return
			}

			// Save the restored config
			if err := configService.SaveFullConfig(restoredCfg); err != nil {
				respondError(w, http.StatusInternalServerError, "CONFIG_SAVE_FAILED", err.Error())
				return
			}

			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("config.restore", claims.Username, "config restored from snapshot "+snapshotID)
				}
			}

			respondOK(w, map[string]any{
				"restoredSnapshotId":   snapshotID,
				"preventiveSnapshotId": preventiveSnapshot.ID,
			})
		})

		// F.10: POST /api/config/import — Import config from settings.js content
		r.Post("/api/config/import", func(w http.ResponseWriter, r *http.Request) {
			var payload struct {
				Content string `json:"content"`
			}
			if err := decodeJSON(r, &payload); err != nil {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
				return
			}

			importedCfg, unrecognized, err := configService.ImportConfig(payload.Content)
			if err != nil {
				respondError(w, http.StatusBadRequest, "CONFIG_IMPORT_FAILED", err.Error())
				return
			}

			respondOK(w, map[string]any{
				"config":       importedCfg,
				"unrecognized": unrecognized,
			})
		})

		r.Get("/api/environment", func(w http.ResponseWriter, r *http.Request) {
			state, err := managedEnvService.Load()
			if err != nil {
				respondError(w, http.StatusInternalServerError, "ENV_LOAD_FAILED", err.Error())
				return
			}
			respondOK(w, state)
		})

		r.Post("/api/environment/apply", func(w http.ResponseWriter, r *http.Request) {
			var payload struct {
				Variables []model.ManagedEnvVar `json:"variables"`
			}
			if err := decodeJSON(r, &payload); err != nil {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
				return
			}

			state, err := managedEnvService.Apply(payload.Variables)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "ENV_APPLY_FAILED", err.Error())
				return
			}
			if errors := managedEnvService.Validate(payload.Variables); len(errors) > 0 {
				respondError(w, http.StatusBadRequest, "ENV_INVALID", strings.Join(errors, "; "))
				return
			}

			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("environment.apply", claims.Username, "managed environment updated")
				}
			}

			respondOK(w, state)
		})

		r.Get("/api/backups", func(w http.ResponseWriter, r *http.Request) {
			backups, err := backupService.List()
			if err != nil {
				respondError(w, http.StatusInternalServerError, "BACKUPS_LIST_FAILED", err.Error())
				return
			}
			respondOK(w, backups)
		})

		r.Post("/api/backups/create", func(w http.ResponseWriter, r *http.Request) {
			backup, err := backupService.Create("manual")
			if err != nil {
				respondError(w, http.StatusInternalServerError, "BACKUP_CREATE_FAILED", err.Error())
				return
			}

			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("backup.create", claims.Username, "manual backup created")
				}
			}

			respondOK(w, backup)
		})

		r.Post("/api/backups/{id}/restore", func(w http.ResponseWriter, r *http.Request) {
			backupID := strings.TrimSpace(chi.URLParam(r, "id"))
			if backupID == "" {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "backup id is required")
				return
			}

			release, err := operationLock.Acquire("restoring", backupID)
			if err != nil {
				respondError(w, http.StatusConflict, "OPERATION_LOCKED", err.Error())
				return
			}
			defer release()

			preventive, err := backupService.Restore(backupID, runtimeManager)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "BACKUP_RESTORE_FAILED", err.Error())
				return
			}

			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("backup.restore", claims.Username, "backup restored with preventive backup")
				}
			}

			respondOK(w, map[string]any{
				"restoredBackupId":   backupID,
				"preventiveBackupId": preventive.ID,
			})
		})

		r.Get("/api/libraries", func(w http.ResponseWriter, r *http.Request) {
			libraries, err := libraryService.List()
			if err != nil {
				respondError(w, http.StatusInternalServerError, "LIBRARIES_LIST_FAILED", err.Error())
				return
			}
			respondOK(w, libraries)
		})

		r.Get("/api/flows", func(w http.ResponseWriter, r *http.Request) {
			flows, err := flowService.List()
			if err != nil {
				respondError(w, http.StatusInternalServerError, "FLOWS_LIST_FAILED", err.Error())
				return
			}
			respondOK(w, flows)
		})

		r.Get("/api/flows/{id}", func(w http.ResponseWriter, r *http.Request) {
			flowID := strings.TrimSpace(chi.URLParam(r, "id"))
			if flowID == "" {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "flow id is required")
				return
			}

			flow, err := flowService.Get(flowID)
			if err != nil {
				if errors.Is(err, service.ErrFlowNotFound) {
					respondError(w, http.StatusNotFound, "FLOW_NOT_FOUND", "flow not found")
					return
				}
				respondError(w, http.StatusInternalServerError, "FLOW_DETAIL_FAILED", err.Error())
				return
			}

			respondOK(w, flow)
		})

		r.Post("/api/libraries/{name}", func(w http.ResponseWriter, r *http.Request) {
			name := strings.TrimSpace(chi.URLParam(r, "name"))
			if name == "" {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "package name is required")
				return
			}

			release, err := operationLock.Acquire("installing", name)
			if err != nil {
				respondError(w, http.StatusConflict, "OPERATION_LOCKED", err.Error())
				return
			}
			defer release()

			result, err := libraryService.Install(name)
			if err != nil {
				respondError(w, http.StatusBadRequest, "LIBRARY_INSTALL_FAILED", err.Error())
				return
			}

			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("library.install", claims.Username, "npm library installed")
				}
			}

			respondOK(w, result)
		})

		r.Delete("/api/libraries/{name}", func(w http.ResponseWriter, r *http.Request) {
			name := strings.TrimSpace(chi.URLParam(r, "name"))
			if name == "" {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "package name is required")
				return
			}

			release, err := operationLock.Acquire("installing", name)
			if err != nil {
				respondError(w, http.StatusConflict, "OPERATION_LOCKED", err.Error())
				return
			}
			defer release()

			result, err := libraryService.Uninstall(name)
			if err != nil {
				respondError(w, http.StatusBadRequest, "LIBRARY_UNINSTALL_FAILED", err.Error())
				return
			}

			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("library.uninstall", claims.Username, "npm library removed")
				}
			}

			respondOK(w, result)
		})

		r.Get("/api/updates/status", func(w http.ResponseWriter, r *http.Request) {
			status, err := updateService.Status()
			if err != nil {
				respondError(w, http.StatusInternalServerError, "UPDATE_STATUS_FAILED", err.Error())
				return
			}
			respondOK(w, status)
		})

		r.Post("/api/updates/apply", func(w http.ResponseWriter, r *http.Request) {
			release, err := operationLock.Acquire("updating", "node-red")
			if err != nil {
				respondError(w, http.StatusConflict, "OPERATION_LOCKED", err.Error())
				return
			}
			defer release()

			result, err := updateService.Apply(runtimeManager)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "UPDATE_APPLY_FAILED", err.Error())
				return
			}

			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("update.apply", claims.Username, "node-red update applied")
				}
			}

			respondOK(w, result)
		})
	})
}

func respondOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.APIResponse[any]{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC(),
	})
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.APIResponse[any]{
		Success: false,
		Error: &model.APIError{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now().UTC(),
	})
}

func respondErrorWithRequest(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	reqID := middleware.GetRequestID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.APIResponse[any]{
		Success: false,
		Error: &model.APIError{
			Code:      code,
			Message:   message,
			RequestID: reqID,
		},
		RequestID: reqID,
		Timestamp: time.Now().UTC(),
	})
}

func respondErrorWithDetails(w http.ResponseWriter, r *http.Request, status int, code, message string, details any) {
	reqID := middleware.GetRequestID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.APIResponse[any]{
		Success: false,
		Error: &model.APIError{
			Code:      code,
			Message:   message,
			RequestID: reqID,
			Details:   details,
		},
		RequestID: reqID,
		Timestamp: time.Now().UTC(),
	})
}

func respondOKWithRequest(w http.ResponseWriter, r *http.Request, data any) {
	reqID := middleware.GetRequestID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(model.APIResponse[any]{
		Success:   true,
		Data:      data,
		RequestID: reqID,
		Timestamp: time.Now().UTC(),
	})
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeSessionCookie(w http.ResponseWriter, r *http.Request, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     service.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: sessionCookieSameSite,
		Secure:   isSecureRequest(r),
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     service.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: sessionCookieSameSite,
		Secure:   isSecureRequest(r),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}

func requireAuth(authService *service.AuthService) func(http.Handler) http.Handler {
	return middleware.RequireAuth(authService)
}

func requireCSRF(authService *service.AuthService) func(http.Handler) http.Handler {
	return middleware.RequireCSRF(authService)
}

func requestClientAddress(r *http.Request) string {
	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	return r.RemoteAddr
}
