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

func TestSetup_TokenNotConsumedOnUserCreationFailure(t *testing.T) {
	h, authSvc, tokenPath, token := newRecoverySetupHandler(t)
	consumeCalls := 0
	h.consumeSetupToken = func(path string) error {
		consumeCalls++
		return setupstate.ConsumeTokenFile(path)
	}

	failed := performSetupRequest(h, "existing@example.com", token.Raw)
	if failed.Code != http.StatusInternalServerError {
		t.Fatalf("failed setup status=%d want=%d body=%s", failed.Code, http.StatusInternalServerError, failed.Body.String())
	}
	if consumeCalls != 0 {
		t.Fatalf("token consumed before user creation completed: calls=%d", consumeCalls)
	}
	if _, err := os.Stat(tokenPath); err != nil {
		t.Fatalf("token missing after user creation failure: %v", err)
	}

	retry := performSetupRequest(h, "recovery@example.com", token.Raw)
	if retry.Code != http.StatusCreated {
		t.Fatalf("retry status=%d want=%d body=%s", retry.Code, http.StatusCreated, retry.Body.String())
	}
	if consumeCalls != 1 {
		t.Fatalf("successful retry consume calls=%d want=1", consumeCalls)
	}
	if authSvc.GetUserByUsername("recovery@example.com") == nil {
		t.Fatal("successful retry did not create the recovery user")
	}
}

func TestSetup_TokenConsumedOnlyAfterUserSuccess(t *testing.T) {
	h, authSvc, tokenPath, token := newRecoverySetupHandler(t)
	consumeCalls := 0
	h.consumeSetupToken = func(path string) error {
		consumeCalls++
		if authSvc.GetUserByUsername("recovery@example.com") == nil {
			t.Fatal("token consumed before recovery user was persisted")
		}
		return setupstate.ConsumeTokenFile(path)
	}

	response := performSetupRequest(h, "recovery@example.com", token.Raw)
	if response.Code != http.StatusCreated {
		t.Fatalf("setup status=%d want=%d body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	if consumeCalls != 1 {
		t.Fatalf("consume calls=%d want=1", consumeCalls)
	}
	if _, err := os.Stat(tokenPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("token file still exists after successful setup: %v", err)
	}
}

func TestSetup_TokenNotConsumedOnRefreshSessionFailure(t *testing.T) {
	h, _, tokenPath, token := newRecoverySetupHandler(t)
	h.createRefreshSession = func(string) (string, error) {
		return "", errors.New("forced refresh-session failure")
	}

	response := performSetupRequest(h, "recovery@example.com", token.Raw)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("setup status=%d want=%d body=%s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
	if _, err := os.Stat(tokenPath); err != nil {
		t.Fatalf("token missing after refresh-session failure: %v", err)
	}
}

func TestSetup_TokenConsumeFailureDoesNotFailSuccessfulSetup(t *testing.T) {
	h, authSvc, tokenPath, token := newRecoverySetupHandler(t)
	h.consumeSetupToken = func(string) error { return errors.New("forced consume failure") }

	response := performSetupRequest(h, "recovery@example.com", token.Raw)
	if response.Code != http.StatusCreated {
		t.Fatalf("setup status=%d want=%d body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	if authSvc.GetUserByUsername("recovery@example.com") == nil {
		t.Fatal("recovery user was not persisted")
	}
	if _, err := os.Stat(tokenPath); err != nil {
		t.Fatalf("failed consume unexpectedly removed token: %v", err)
	}
}

func TestSetup_TokenReplayRejected(t *testing.T) {
	h, authSvc, _, token := newRecoverySetupHandler(t)
	first := performSetupRequest(h, "recovery@example.com", token.Raw)
	if first.Code != http.StatusCreated {
		t.Fatalf("first setup status=%d want=%d body=%s", first.Code, http.StatusCreated, first.Body.String())
	}

	replay := performSetupRequest(h, "replayed@example.com", token.Raw)
	if replay.Code != http.StatusConflict {
		t.Fatalf("replay status=%d want=%d body=%s", replay.Code, http.StatusConflict, replay.Body.String())
	}
	users, err := authSvc.GetAllUsers()
	if err != nil || len(users) != 2 {
		t.Fatalf("users after replay=%d want=2 err=%v", len(users), err)
	}
}

func TestSetup_ConcurrentRecoveryConsumesTokenOnce(t *testing.T) {
	h, authSvc, _, token := newRecoverySetupHandler(t)
	const requests = 5
	statuses := make([]int, requests)
	runConcurrentHandler(t, requests, func(i int) {
		response := performSetupRequest(h, "recovery-"+string(rune('a'+i))+"@example.com", token.Raw)
		statuses[i] = response.Code
	})

	created, conflicts := 0, 0
	for _, status := range statuses {
		switch status {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("unexpected concurrent setup status: %d", status)
		}
	}
	if created != 1 || conflicts != requests-1 {
		t.Fatalf("concurrent outcomes: created=%d conflicts=%d", created, conflicts)
	}
	users, err := authSvc.GetAllUsers()
	if err != nil || len(users) != 2 {
		t.Fatalf("users after concurrent recovery=%d want=2 err=%v", len(users), err)
	}
}

func newRecoverySetupHandler(t *testing.T) (*AuthHandler, *service.AuthService, string, setupstate.SetupToken) {
	t.Helper()
	dir := t.TempDir()
	authSvc := service.NewAuthService("test-secret",
		store.NewJSONStore[model.CCUsers](filepath.Join(dir, "cc-users.json")),
		store.NewJSONStore[model.RefreshSessions](filepath.Join(dir, "sessions.json")))
	now := model.NowISO8601()
	if err := authSvc.CreateUser(&model.CCUser{ID: "existing", Username: "existing@example.com", PasswordHash: "hash", Role: model.RoleAdmin, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	token, err := setupstate.GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	tokenPath := filepath.Join(dir, setupstate.SetupTokenFileName)
	if err := setupstate.WriteTokenFile(tokenPath, token); err != nil {
		t.Fatal(err)
	}
	h := NewAuthHandler(authSvc)
	h.SetSetupTokenPath(tokenPath)
	return h, authSvc, tokenPath, token
}

func performSetupRequest(h *AuthHandler, username, token string) *httptest.ResponseRecorder {
	body := `{"username":"` + username + `","password":"correct-horse-battery-staple"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/setup", strings.NewReader(body))
	req.Header.Set("X-Setup-Reset-Token", token)
	rec := httptest.NewRecorder()
	h.Setup(rec, req)
	return rec
}

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
