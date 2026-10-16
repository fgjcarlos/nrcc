package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/go-chi/chi/v5"
)

// isolateAuthzHandlers rebinds only terminal ChainHandlers on the existing mux.
// In server.go, Auth and the baseline throttle are inherited from the protected
// group; endpoint-specific throttles and RequireAdmin follow them in the chain.
// Reusing that exact chain puts the sentinel AFTER every registered gate.
// Unchained handlers (including the SPA fallback) stay intact, so a missing API
// route cannot masquerade as a successful gate probe via the catch-all.
func isolateAuthzHandlers(t *testing.T, mux *chi.Mux) {
	t.Helper()
	for _, route := range mux.Routes() {
		if route.SubRoutes != nil {
			child, ok := route.SubRoutes.(*chi.Mux)
			if !ok {
				t.Fatalf("route %s: unsupported subrouter %T", route.Pattern, route.SubRoutes)
			}
			isolateAuthzHandlers(t, child)
			continue
		}
		for method, endpoint := range route.Handlers {
			chain, ok := endpoint.(*chi.ChainHandler)
			if !ok {
				continue
			}
			sentinel := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("X-Authz-Gate", "passed")
				w.WriteHeader(http.StatusNoContent)
			})
			mux.Method(method, route.Pattern, chi.Chain(chain.Middlewares...).Handler(sentinel))
		}
	}
}

// Authentication must still run before the sentinel on every protected probe.
func TestAuthz_UnauthenticatedCannotReachGate(t *testing.T) {
	srv, _ := newAuthzGateTestServer(t)
	for _, route := range mutatingRoutes {
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, httptest.NewRequest(route.method, route.path, nil))
		if rec.Code != http.StatusUnauthorized || rec.Header().Get("X-Authz-Gate") != "" {
			t.Errorf("unauthenticated %s %s: status=%d marker=%q; want 401 without sentinel",
				route.method, route.path, rec.Code, rec.Header().Get("X-Authz-Gate"))
		}
	}
}

// rejectedAuthzBackup records entry into the update flow without creating an
// archive or allowing the flow to reach an installer, even if isolation breaks.
type rejectedAuthzBackup struct {
	called chan struct{}
}

func (b rejectedAuthzBackup) CreateTyped(model.BackupType, string) (model.Backup, error) {
	b.called <- struct{}{}
	return model.Backup{}, errors.New("authorization tests must not create update backups")
}

func TestAuthz_UpdateApplyDoesNotStartUpdate(t *testing.T) {
	srv, authSvc := newAuthzGateTestServer(t)
	backup := rejectedAuthzBackup{called: make(chan struct{}, 1)}
	srv.updateSvc.SetBackupCreator(backup)

	req := httptest.NewRequest(http.MethodPost, "/api/updates/apply", nil)
	req.Header.Set("Authorization", "Bearer "+tokenForRole(t, authSvc, "admin", model.RoleAdmin))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		// The real handler responds before its detached goroutine runs. Join
		// the backup probe for a deterministic diagnosis, not a cleanup race.
		select {
		case <-backup.called:
			t.Errorf("authorization request entered the update backup flow")
		case <-time.After(5 * time.Second):
			t.Errorf("authorization request did not reach the gate sentinel")
		}
		t.Fatalf("admin update gate: status=%d body=%q; want 204 and empty body", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Errorf("admin update gate body = %q; want empty", rec.Body.String())
	}
	select {
	case <-backup.called:
		t.Error("authorization request entered the update backup flow")
	default:
	}
	if state := srv.updateSvc.GetFlowState(); state.State != model.StateIdle {
		t.Errorf("update state = %s; want Idle", state.State)
	}
	if history := srv.updateSvc.History(); len(history) != 0 {
		t.Errorf("update history = %v; want empty", history)
	}
}
