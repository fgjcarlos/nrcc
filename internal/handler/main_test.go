package handler

import (
	"os"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/service"
	"golang.org/x/crypto/bcrypt"
)

// TestMain lowers bcrypt cost for the entire handler package test suite.
// Handler tests exercise auth flows that internally call service.CreateUser
// and MfaService.ConfirmEnrollment, both of which run bcrypt at cost 12 in
// production. With ~40+ tests running serially the suite exceeded the 10-min
// CI timeout (see audit HIGH-005); switching to MinCost restores headroom.
func TestMain(m *testing.M) {
	service.SetBcryptCostForTest(bcrypt.MinCost)
	os.Exit(m.Run())
}