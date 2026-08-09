package service

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestMain lowers bcrypt cost for the entire service package test suite so
// the ~40+ tests that hash/verify passwords (auth, mfa, password) complete
// inside the 10-minute CI timeout (see audit HIGH-005).
func TestMain(m *testing.M) {
	SetBcryptCostForTest(bcrypt.MinCost)
	os.Exit(m.Run())
}