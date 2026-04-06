package server

import (
	"context"
	"encoding/json"
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
	Port       string
	Frontend   fs.FS
	Runtime    *service.ProcessManager
	Auth       *service.AuthService
	Config     service.ConfigService
	ManagedEnv service.ManagedEnvService
	Backups    service.BackupService
}

type Server struct {
	httpServer *http.Server
}

type authResponse struct {
	User      *model.UserPublic `json:"user"`
	CSRFToken string            `json:"csrfToken"`
}

func New(cfg Config) *Server {
	router := chi.NewRouter()
	registerAPIRoutes(router, cfg.Runtime, cfg.Auth, cfg.Config, cfg.ManagedEnv, cfg.Backups)
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

func registerAPIRoutes(router chi.Router, runtimeManager *service.ProcessManager, authService *service.AuthService, configService service.ConfigService, managedEnvService service.ManagedEnvService, backupService service.BackupService) {
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
			respondOK(w, map[string]any{
				"goos":      runtime.GOOS,
				"goarch":    runtime.GOARCH,
				"cpus":      runtime.NumCPU(),
				"hostname":  hostname,
				"timestamp": time.Now().UTC(),
			})
		})

		r.Get("/api/config", func(w http.ResponseWriter, r *http.Request) {
			cfg, err := configService.Load()
			if err != nil {
				respondError(w, http.StatusInternalServerError, "CONFIG_LOAD_FAILED", err.Error())
				return
			}
			respondOK(w, cfg)
		})

		r.Post("/api/config/validate", func(w http.ResponseWriter, r *http.Request) {
			var payload model.AppConfig
			if err := decodeJSON(r, &payload); err != nil {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
				return
			}
			respondOK(w, configService.Validate(payload))
		})

		r.Post("/api/config/apply", func(w http.ResponseWriter, r *http.Request) {
			var payload model.AppConfig
			if err := decodeJSON(r, &payload); err != nil {
				respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
				return
			}

			result, err := configService.Apply(payload)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "CONFIG_APPLY_FAILED", err.Error())
				return
			}
			if !result.Valid {
				respondError(w, http.StatusBadRequest, "CONFIG_INVALID", strings.Join(result.Errors, "; "))
				return
			}

			if authService != nil {
				if claims, ok := middleware.AuthClaimsFromContext(r.Context()); ok {
					authService.LogAudit("config.apply", claims.Username, "supported config applied")
				}
			}

			respondOK(w, result)
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
		SameSite: http.SameSiteStrictMode,
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
		SameSite: http.SameSiteStrictMode,
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
