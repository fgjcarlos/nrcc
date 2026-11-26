package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func TestLogin_UnknownUserStillVerifiesPassword(t *testing.T) {
	h, svc := setupAuthTest(t)
	knownHash := svc.GetUserByUsername("admin").PasswordHash
	for _, name := range []string{"unknown-user", "admin"} {
		t.Run(name, func(t *testing.T) {
			var hashes []string
			h.verifyLoginPassword = func(hash, password string) bool {
				hashes = append(hashes, hash)
				return name == "unknown-user" // Must fail even if the dummy hash matches.
			}
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"`+name+`","password":"wrong-password"}`))
			res := httptest.NewRecorder()
			h.Login(res, req)
			var body model.ApiResponse[AuthResponse]
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if res.Code != http.StatusUnauthorized || body.Error == nil || body.Error.Code != "AUTH_FAILED" || body.Error.Message != "Invalid username or password" || len(res.Result().Cookies()) != 0 {
				t.Fatalf("unexpected rejected login: HTTP %d, error %+v", res.Code, body.Error)
			}
			if len(hashes) != 1 {
				t.Fatalf("password checks = %d, want 1", len(hashes))
			}
			if name == "admin" {
				if hashes[0] != knownHash {
					t.Fatal("known account was not checked against its own hash")
				}
				return
			}
			cost, err := bcrypt.Cost([]byte(hashes[0]))
			if err != nil || cost != service.ProductionBcryptCost || hashes[0] == knownHash {
				t.Fatal("unknown account was not checked against a production-cost dummy bcrypt hash")
			}
		})
	}
}
