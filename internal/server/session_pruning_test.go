package server

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
)

func TestServerPrunesRefreshSessionsOnStartup(t *testing.T) {
	dir := t.TempDir()
	users := store.NewJSONStore[model.CCUsers](filepath.Join(dir, "users.json"))
	sessions := store.NewJSONStore[model.RefreshSessions](filepath.Join(dir, "refresh_sessions.json"))
	auth := service.NewAuthService("test-secret", users, sessions)
	if err := auth.CreateUser(&model.CCUser{ID: "user", Username: "user", Role: model.RoleAdmin}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	active := model.RefreshSession{ID: "active", ExpiresAt: now.Add(time.Hour).Unix()}
	if err := sessions.Write(model.RefreshSessions{Sessions: []model.RefreshSession{
		{ID: "stale", ExpiresAt: now.Add(-25 * time.Hour).Unix()}, active,
	}}); err != nil {
		t.Fatal(err)
	}

	srv := NewServerWithConfig(auth, Config{DataDir: dir, HTTPLogger: discardHTTPLogger()})
	t.Cleanup(srv.Shutdown)
	got, err := sessions.Read()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Sessions, []model.RefreshSession{active}) {
		t.Fatalf("startup left unexpected sessions: %+v", got.Sessions)
	}
}
