package service

import (
	"testing"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func TestValidatePassword_MinLength(t *testing.T) {
	if err := ValidatePassword("short"); err == nil {
		t.Error("should reject passwords shorter than 8 chars")
	}
	if err := ValidatePassword("g00d-pwd"); err != nil {
		t.Errorf("8-char password should pass length check, got: %v", err)
	}
}

func TestValidatePassword_MaxLength(t *testing.T) {
	long := make([]byte, 73)
	for i := range long {
		long[i] = 'a'
	}
	if err := ValidatePassword(string(long)); err == nil {
		t.Error("should reject passwords longer than 72 chars")
	}
}

func TestValidatePassword_CommonPasswords(t *testing.T) {
	common := []string{"password", "12345678", "admin123", "qwerty123", "PASSWORD"}
	for _, pw := range common {
		if err := ValidatePassword(pw); err == nil {
			t.Errorf("should reject common password %q", pw)
		}
	}
}

func TestValidatePassword_AcceptsStrong(t *testing.T) {
	strong := []string{
		"my-secure-passphrase",
		"Tr0ub4dor&3!",
		"correct-horse-battery",
	}
	for _, pw := range strong {
		if err := ValidatePassword(pw); err != nil {
			t.Errorf("should accept %q, got: %v", pw, err)
		}
	}
}

func TestHashPassword_UsesCost12(t *testing.T) {
	tempDir := t.TempDir()
	userStore := store.NewJSONStore[model.CCUsers](tempDir + "/users.json")
	sessionStore := store.NewJSONStore[model.RefreshSessions](tempDir + "/sessions.json")
	authSvc := NewAuthService("test-secret", userStore, sessionStore)

	hash, err := authSvc.HashPassword("test-password-12345")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("bcrypt.Cost error: %v", err)
	}

	if cost != BcryptCost {
		t.Errorf("bcrypt cost = %d, want %d", cost, BcryptCost)
	}
}

func TestNeedsRehash_LowerCost(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("test"), 10)
	if !NeedsRehash(string(hash)) {
		t.Error("cost 10 hash should need rehash to 12")
	}
}

func TestNeedsRehash_CurrentCost(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("test"), BcryptCost)
	if NeedsRehash(string(hash)) {
		t.Error("cost 12 hash should not need rehash")
	}
}
