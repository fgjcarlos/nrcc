package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLog_WritesJSONLEvent(t *testing.T) {
	svc, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close()

	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	req.RemoteAddr = "192.168.1.10:12345"
	req.Header.Set("User-Agent", "TestAgent/1.0")

	svc.Log(req, "admin", "LOGIN", "", "ok", map[string]string{"method": "password"})

	svc.Close()

	data, err := os.ReadFile(filepath.Join(svc.dir, fileName))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, data)
	}

	if event.Actor != "admin" {
		t.Errorf("Actor = %q, want %q", event.Actor, "admin")
	}
	if event.Action != "LOGIN" {
		t.Errorf("Action = %q, want %q", event.Action, "LOGIN")
	}
	if event.Result != "ok" {
		t.Errorf("Result = %q, want %q", event.Result, "ok")
	}
	if event.IP != "192.168.1.10" {
		t.Errorf("IP = %q, want %q", event.IP, "192.168.1.10")
	}
}

func TestLog_NilServiceIsNoop(t *testing.T) {
	var svc *Service
	req := httptest.NewRequest("GET", "/", nil)
	svc.Log(req, "x", "X", "", "ok", nil)
}

func TestLog_MultipleEvents(t *testing.T) {
	svc, _ := NewService(t.TempDir())
	defer svc.Close()

	req := httptest.NewRequest("POST", "/test", nil)

	for i := 0; i < 5; i++ {
		svc.Log(req, "user", fmt.Sprintf("ACTION_%d", i), "", "ok", nil)
	}

	svc.Close()

	f, _ := os.Open(filepath.Join(svc.dir, fileName))
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
	}
	if count != 5 {
		t.Errorf("expected 5 lines, got %d", count)
	}
}

func TestLog_Rotation(t *testing.T) {
	dir := t.TempDir()
	svc, _ := NewService(dir)
	defer svc.Close()

	req := httptest.NewRequest("POST", "/test", nil)
	bigMeta := map[string]string{"data": strings.Repeat("x", 1024)}

	for i := 0; i < 12000; i++ {
		svc.Log(req, "user", "BULK", "", "ok", bigMeta)
	}

	svc.Close()

	entries, _ := os.ReadDir(filepath.Join(dir, "audit"))
	jsonlCount := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			jsonlCount++
		}
	}

	if jsonlCount < 2 {
		t.Errorf("expected at least 2 jsonl files after rotation, got %d", jsonlCount)
	}
}

func TestLog_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	svc, _ := NewService(dir)
	defer svc.Close()

	req := httptest.NewRequest("POST", "/test", nil)
	svc.Log(req, "user", "TEST", "", "ok", nil)

	info, err := os.Stat(filepath.Join(dir, "audit", fileName))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("permissions = %o, want 0600", perm)
	}
}

func TestLog_AuditDirPermissions(t *testing.T) {
	dir := t.TempDir()
	_, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "audit"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("dir permissions = %o, want 0700", perm)
	}
}

func TestLog_XForwardedFor(t *testing.T) {
	svc, _ := NewService(t.TempDir())
	defer svc.Close()

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 172.16.0.1")

	svc.Log(req, "user", "TEST", "", "ok", nil)
	svc.Close()

	data, _ := os.ReadFile(filepath.Join(svc.dir, fileName))
	var event Event
	json.Unmarshal(data, &event)

	if event.IP != "10.0.0.1" {
		t.Errorf("IP = %q, want %q (first in X-Forwarded-For)", event.IP, "10.0.0.1")
	}
}

func TestLog_SecretsNeverLogged(t *testing.T) {
	svc, _ := NewService(t.TempDir())
	defer svc.Close()

	req := httptest.NewRequest("POST", "/test", nil)
	svc.Log(req, "admin", "ENV_SET", "DB_PASS", "ok", map[string]string{
		"key":  "DB_PASS",
		"type": "secret",
	})
	svc.Close()

	data, _ := os.ReadFile(filepath.Join(svc.dir, fileName))
	raw := string(data)

	if strings.Contains(raw, "password") || strings.Contains(raw, "secret-value") {
		t.Error("audit log should never contain secret values")
	}
}
