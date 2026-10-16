package workflow

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Keep the package-exposure evidence behind the GO-2026-5932 disposition
// current. This is not a blanket exemption for the crypto module.
func TestDeliveredMainExcludesOpenPGP(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", ".")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list delivered main dependencies: %v\n%s", err, out)
	}
	foundCrypto := false
	for _, pkg := range strings.Fields(string(out)) {
		if strings.HasPrefix(pkg, "golang.org/x/crypto/") {
			foundCrypto = true
		}
		if pkg == "golang.org/x/crypto/openpgp" || strings.HasPrefix(pkg, "golang.org/x/crypto/openpgp/") {
			t.Fatalf("OpenPGP entered delivered main: %s; reassess GO-2026-5932", pkg)
		}
	}
	if !foundCrypto {
		t.Fatal("expected crypto dependency; reassess exposure probe")
	}
}
