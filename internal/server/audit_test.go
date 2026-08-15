package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitAuditService(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "audit"), []byte("obstruction"), 0600); err != nil {
		t.Fatalf("obstruct audit path: %v", err)
	}

	var reports []string
	svc := initAuditService(dataDir, func(format string, args ...any) {
		reports = append(reports, fmt.Sprintf(format, args...))
	})
	if svc != nil {
		t.Fatal("initAuditService returned an available service for an obstructed path")
	}
	if len(reports) != 1 {
		t.Fatalf("reports = %d, want 1", len(reports))
	}
	if !strings.Contains(reports[0], "audit initialization failed") || !strings.Contains(reports[0], "create audit dir") {
		t.Fatalf("report = %q, want localized audit initialization context", reports[0])
	}
}
