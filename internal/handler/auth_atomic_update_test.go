package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/middleware"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func runConcurrentHandler(t *testing.T, n int, fn func(i int)) {
	t.Helper()
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			fn(i)
		}(i)
	}
	close(start)
	wg.Wait()
}

func newEmptyAtomicAuthHandler(t *testing.T) (*AuthHandler, *service.AuthService) {
	t.Helper()
	dir := t.TempDir()
	authSvc := service.NewAuthService(
		"test-secret",
		store.NewJSONStore[model.CCUsers](dir+"/users.json"),
		store.NewJSONStore[model.RefreshSessions](dir+"/sessions.json"),
	)
	return NewAuthHandler(authSvc), authSvc
}

func requestWithClaims(method, path string, body []byte, claims *model.Claims) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if claims != nil {
		req = req.WithContext(context.WithValue(req.Context(), middleware.CtxKeyUser, claims))
	}
	return req
}

func TestAuthHandler_Setup_Concurrent(t *testing.T) {
	h, authSvc := newEmptyAtomicAuthHandler(t)
	statuses := make([]int, 3)
	runConcurrentHandler(t, len(statuses), func(i int) {
		body, _ := json.Marshal(SetupRequest{Username: fmt.Sprintf("admin-%d", i), Password: "Unique-Setup-Password-42"})
		rec := httptest.NewRecorder()
		h.Setup(rec, requestWithClaims(http.MethodPost, "/api/auth/setup", body, nil))
		statuses[i] = rec.Code
	})
	created, conflicts := 0, 0
	for _, status := range statuses {
		switch status {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("unexpected setup status: %d", status)
		}
	}
	if created != 1 || conflicts != 2 {
		t.Fatalf("setup outcomes: created=%d conflicts=%d", created, conflicts)
	}
	users, err := authSvc.GetAllUsers()
	if err != nil || len(users) != 1 || users[0].Role != model.RoleAdmin {
		t.Fatalf("unexpected users after setup: users=%+v err=%v", users, err)
	}

	t.Run("duplicate create maps to conflict", func(t *testing.T) {
		claims := &model.Claims{UserID: users[0].ID, Username: users[0].Username, Role: model.RoleAdmin}
		statuses := make([]int, 2)
		runConcurrentHandler(t, len(statuses), func(i int) {
			body, _ := json.Marshal(CreateUserRequest{Username: "duplicate", Password: "Unique-New-User-42", Role: model.RoleViewer})
			rec := httptest.NewRecorder()
			h.CreateUser(rec, requestWithClaims(http.MethodPost, "/api/auth/users", body, claims))
			statuses[i] = rec.Code
		})
		created, conflicts := 0, 0
		for _, status := range statuses {
			switch status {
			case http.StatusCreated:
				created++
			case http.StatusConflict:
				conflicts++
			default:
				t.Fatalf("unexpected create status: %d", status)
			}
		}
		if created != 1 || conflicts != 1 {
			t.Fatalf("create outcomes: created=%d conflicts=%d", created, conflicts)
		}
	})
}

func TestAuthHandler_LoginPasswordRehash_Concurrent(t *testing.T) {
	h, authSvc := newEmptyAtomicAuthHandler(t)
	lowCostHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword: %v", err)
	}
	if err := authSvc.CreateUser(&model.CCUser{ID: "u1", Username: "admin", PasswordHash: string(lowCostHash), Role: model.RoleAdmin}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	statuses := make([]int, 3)
	runConcurrentHandler(t, len(statuses), func(i int) {
		rec := httptest.NewRecorder()
		h.Login(rec, requestWithClaims(http.MethodPost, "/api/auth/login", []byte(`{"username":"admin","password":"password123"}`), nil))
		statuses[i] = rec.Code
	})
	for _, status := range statuses {
		if status != http.StatusOK {
			t.Fatalf("login status=%d, want 200", status)
		}
	}
	user := authSvc.GetUserByID("u1")
	if user == nil {
		t.Fatal("user disappeared after login rehash")
	}
	cost, err := bcrypt.Cost([]byte(user.PasswordHash))
	if err != nil || cost != service.BcryptCost {
		t.Fatalf("rehash cost=%d err=%v, want %d", cost, err, service.BcryptCost)
	}
}

func TestAuthHandler_DeleteUser_Concurrent(t *testing.T) {
	h, authSvc := newEmptyAtomicAuthHandler(t)
	for _, user := range []*model.CCUser{
		{ID: "admin", Username: "admin", Role: model.RoleAdmin},
		{ID: "target", Username: "target", Role: model.RoleViewer},
	} {
		if err := authSvc.CreateUser(user); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
	}
	claims := &model.Claims{UserID: "admin", Username: "admin", Role: model.RoleAdmin}
	statuses := make([]int, 10)
	runConcurrentHandler(t, len(statuses), func(i int) {
		req := requestWithClaims(http.MethodDelete, "/api/auth/users/target", nil, claims)
		req.SetPathValue("id", "target")
		rec := httptest.NewRecorder()
		h.DeleteUser(rec, req)
		statuses[i] = rec.Code
	})
	ok, notFound := 0, 0
	for _, status := range statuses {
		switch status {
		case http.StatusOK:
			ok++
		case http.StatusNotFound:
			notFound++
		default:
			t.Fatalf("unexpected delete status: %d", status)
		}
	}
	if ok != 1 || notFound != len(statuses)-1 {
		t.Fatalf("delete outcomes: ok=%d notFound=%d", ok, notFound)
	}
}

func TestAuthHandler_ChangePassword_Concurrent(t *testing.T) {
	h, authSvc := newEmptyAtomicAuthHandler(t)
	if err := authSvc.CreateUser(&model.CCUser{ID: "u1", Username: "admin", PasswordHash: "old", Role: model.RoleAdmin}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	passwords := []string{"password-one", "password-two", "password-three"}
	statuses := make([]int, len(passwords))
	claims := &model.Claims{UserID: "u1", Username: "admin", Role: model.RoleAdmin}
	runConcurrentHandler(t, len(passwords), func(i int) {
		body, _ := json.Marshal(PasswordChangeRequest{Password: passwords[i]})
		req := requestWithClaims(http.MethodPatch, "/api/auth/users/u1/password", body, claims)
		req.SetPathValue("id", "u1")
		rec := httptest.NewRecorder()
		h.ChangePassword(rec, req)
		statuses[i] = rec.Code
	})
	for _, status := range statuses {
		if status != http.StatusOK {
			t.Fatalf("change-password status=%d, want 200", status)
		}
	}
	user := authSvc.GetUserByID("u1")
	matches := 0
	for _, password := range passwords {
		if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) == nil {
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("final password hash matched %d proposed passwords", matches)
	}
}

func TestAuthHandler_UpdateUser_Concurrent(t *testing.T) {
	h, authSvc := newEmptyAtomicAuthHandler(t)
	for _, user := range []*model.CCUser{
		{ID: "admin-1", Username: "admin-1", Role: model.RoleAdmin},
		{ID: "admin-2", Username: "admin-2", Role: model.RoleAdmin},
	} {
		if err := authSvc.CreateUser(user); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
	}
	claims := &model.Claims{UserID: "operator", Username: "operator", Role: model.RoleAdmin}
	statuses := make([]int, 2)
	runConcurrentHandler(t, 2, func(i int) {
		role := model.RoleViewer
		id := fmt.Sprintf("admin-%d", i+1)
		body, _ := json.Marshal(UpdateUserRequest{Role: &role})
		req := requestWithClaims(http.MethodPatch, "/api/auth/users/"+id, body, claims)
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		h.UpdateUser(rec, req)
		statuses[i] = rec.Code
	})
	ok, forbidden := 0, 0
	for _, status := range statuses {
		switch status {
		case http.StatusOK:
			ok++
		case http.StatusForbidden:
			forbidden++
		default:
			t.Fatalf("unexpected update status: %d", status)
		}
	}
	if ok != 1 || forbidden != 1 {
		t.Fatalf("update outcomes: ok=%d forbidden=%d", ok, forbidden)
	}
	users, err := authSvc.GetAllUsers()
	if err != nil {
		t.Fatalf("GetAllUsers: %v", err)
	}
	admins := 0
	for _, user := range users {
		if user.Role == model.RoleAdmin {
			admins++
		}
	}
	if admins != 1 {
		t.Fatalf("admin count=%d, want 1", admins)
	}
}
