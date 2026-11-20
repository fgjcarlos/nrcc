package service

import (
	"os"
	"os/exec"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestMain lowers bcrypt cost for the entire service package test suite so
// the ~40+ tests that hash/verify passwords (auth, mfa, password) complete
// inside the 10-minute CI timeout (see audit HIGH-005).
//
// It also keeps host detection off the real machine: every structured
// settings save runs HostService.Detect, which would otherwise spawn node,
// npm, node-red, portless and docker and make the suite's speed and
// outcome depend on the host. Tests that exercise detection install their
// own execLookPath/execCommand stubs.
func TestMain(m *testing.M) {
	SetBcryptCostForTest(bcrypt.MinCost)
	execLookPath = func(name string) (string, error) {
		return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
	}
	os.Exit(m.Run())
}