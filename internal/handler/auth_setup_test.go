package handler

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	setupstate "github.com/fgjcarlos/nrcc/internal/setup"
	"github.com/fgjcarlos/nrcc/internal/store"
)

func TestSetup_NoUserFirstBoot(t *testing.T)         { runSetupCase(t, "first-boot") }
func TestSetup_UserExists_ValidToken(t *testing.T)   { runSetupCase(t, "valid-token") }
func TestSetup_UserExists_InvalidToken(t *testing.T) { runSetupCase(t, "invalid-token") }
func TestSetup_UserExists_NoHeader(t *testing.T)     { runSetupCase(t, "no-header") }
func TestSetup_NormalizesUsername(t *testing.T)      { runSetupCase(t, "normalize") }

func runSetupCase(t *testing.T, scenario string) {
	t.Helper()
	dir := t.TempDir()
	authSvc := service.NewAuthService("test-secret",
		store.NewJSONStore[model.CCUsers](filepath.Join(dir, "cc-users.json")),
		store.NewJSONStore[model.RefreshSessions](filepath.Join(dir, "sessions.json")))
	configured := scenario == "valid-token" || scenario == "invalid-token" || scenario == "no-header"
	if configured {
		now := model.NowISO8601()
		if err := authSvc.CreateUser(&model.CCUser{ID: "existing", Username: "existing@example.com", PasswordHash: "hash", Role: model.RoleAdmin, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, setupstate.SetupTokenFileName)
	var token setupstate.SetupToken
	if configured {
		var err error
		token, err = setupstate.GenerateToken()
		if err != nil {
			t.Fatal(err)
		}
		if err := setupstate.WriteTokenFile(path, token); err != nil {
			t.Fatal(err)
		}
	}
	h := NewAuthHandler(authSvc)
	h.SetSetupTokenPath(path)
	username := map[string]string{"normalize": " Admin@Example.com "}[scenario]
	if username == "" {
		username = "recovery@example.com"
	}
	header := map[string]string{"valid-token": token.Raw, "invalid-token": strings.Repeat("0", 64)}[scenario]
	body := `{"username":"` + username + `","password":"correct-horse-battery-staple"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/setup", strings.NewReader(body))
	req.Header.Set("X-Setup-Reset-Token", header)
	rec := httptest.NewRecorder()
	h.Setup(rec, req)
	wantStatus := http.StatusCreated
	wantUsers := 1
	if scenario == "invalid-token" || scenario == "no-header" {
		wantStatus = http.StatusConflict
	}
	if scenario == "valid-token" {
		wantUsers = 2
	}
	if rec.Code != wantStatus {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, wantStatus, rec.Body.String())
	}
	users, _ := authSvc.GetAllUsers()
	if len(users) != wantUsers {
		t.Fatalf("users=%d want=%d", len(users), wantUsers)
	}
	if scenario == "normalize" && authSvc.GetUserByUsername("admin@example.com") == nil {
		t.Fatal("username was not normalized")
	}
	_, statErr := os.Stat(path)
	if scenario == "valid-token" && !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("token not consumed: %v", statErr)
	}
	if (scenario == "invalid-token" || scenario == "no-header") && statErr != nil {
		t.Fatalf("token consumed: %v", statErr)
	}
}
