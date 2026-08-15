package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

type failingWriteFile struct {
	auditFile
	err error
}

func (f failingWriteFile) Write([]byte) (int, error) { return 0, f.err }

type testFS struct {
	osFileSystem
	wrapOpen func(auditFile) auditFile
	linkErr  error
}

func (f *testFS) OpenFile(name string, flag int, perm os.FileMode) (auditFile, error) {
	file, err := f.osFileSystem.OpenFile(name, flag, perm)
	if err == nil && f.wrapOpen != nil {
		file = f.wrapOpen(file)
	}
	return file, err
}

func (f *testFS) Link(oldname, newname string) error {
	if f.linkErr != nil {
		return f.linkErr
	}
	return f.osFileSystem.Link(oldname, newname)
}

type reportRecorder struct{ reports []Report }

func (r *reportRecorder) Report(report Report) { r.reports = append(r.reports, report) }

func testDependencies(now time.Time, fs fileSystem, reporter Reporter) dependencies {
	return dependencies{now: func() time.Time { return now }, fs: fs, reporter: reporter}
}

func TestService_RotationNames(t *testing.T) {
	now := time.Date(2026, time.August, 15, 12, 34, 56, 123456789, time.UTC)
	dir := t.TempDir()
	svc, err := newService(dir, testDependencies(now, &testFS{}, &reportRecorder{}))
	if err != nil {
		t.Fatalf("newService: %v", err)
	}
	defer func() { _ = svc.Close() }()

	want := []string{
		"audit-20260815-123456.123456789-000000.jsonl",
		"audit-20260815-123456.123456789-000001.jsonl",
	}
	for i, name := range want {
		got, err := svc.selectBackupName()
		if err != nil {
			t.Fatalf("selectBackupName %d: %v", i, err)
		}
		if filepath.Base(got) != name {
			t.Fatalf("candidate %d = %q, want %q", i, filepath.Base(got), name)
		}
		if err := os.WriteFile(got, []byte(fmt.Sprintf("payload-%d", i)), 0600); err != nil {
			t.Fatalf("create candidate %d: %v", i, err)
		}
	}

	first, err := os.ReadFile(filepath.Join(svc.dir, want[0]))
	if err != nil {
		t.Fatalf("read first candidate: %v", err)
	}
	if string(first) != "payload-0" {
		t.Fatalf("existing candidate changed: %q", first)
	}
}

func TestService_LogOutcomes(t *testing.T) {
	writeErr := fmt.Errorf("disk full")
	linkErr := fmt.Errorf("link denied")
	tests := []struct {
		name          string
		configure     func(*Service, *testFS)
		wantPersisted bool
		wantKind      string
		wantStage     string
	}{
		{
			name: "event write failure",
			configure: func(svc *Service, _ *testFS) {
				svc.file = failingWriteFile{auditFile: svc.file, err: writeErr}
			},
			wantKind:  "event",
			wantStage: "write",
		},
		{
			name: "post-write rotation failure",
			configure: func(svc *Service, fs *testFS) {
				svc.size = maxSize - 1
				fs.linkErr = linkErr
			},
			wantPersisted: true,
			wantKind:      "maintenance",
			wantStage:     "link-backup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &testFS{}
			reporter := &reportRecorder{}
			svc, err := newService(t.TempDir(), testDependencies(time.Now(), fs, reporter))
			if err != nil {
				t.Fatalf("newService: %v", err)
			}
			defer func() { _ = svc.Close() }()
			tt.configure(svc, fs)

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			outcome := svc.Log(req, "user", "TEST", "", "ok", nil)
			if outcome.Persisted != tt.wantPersisted {
				t.Fatalf("Persisted = %v, want %v", outcome.Persisted, tt.wantPersisted)
			}
			if len(reporter.reports) != 1 {
				t.Fatalf("reports = %d, want 1", len(reporter.reports))
			}
			if got := reporter.reports[0]; got.Kind != tt.wantKind || got.Stage != tt.wantStage || got.Persisted != tt.wantPersisted {
				t.Fatalf("report = %+v, want kind=%q stage=%q persisted=%v", got, tt.wantKind, tt.wantStage, tt.wantPersisted)
			}
		})
	}
}

func TestLog_WritesJSONLEvent(t *testing.T) {
	svc, err := NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer func() { _ = svc.Close() }()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "192.168.1.10:12345"
	req.Header.Set("User-Agent", "TestAgent/1.0")

	svc.Log(req, "admin", "LOGIN", "", "ok", map[string]string{"method": "password"})
	_ = svc.Close()

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
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	svc.Log(req, "x", "X", "", "ok", nil)
}

func TestLog_MultipleEvents(t *testing.T) {
	svc, _ := NewService(t.TempDir())
	defer func() { _ = svc.Close() }()

	req := httptest.NewRequest(http.MethodPost, "/test", nil)

	for i := 0; i < 5; i++ {
		svc.Log(req, "user", fmt.Sprintf("ACTION_%d", i), "", "ok", nil)
	}

	_ = svc.Close()

	f, _ := os.Open(filepath.Join(svc.dir, fileName))
	defer func() { _ = f.Close() }()

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
	defer func() { _ = svc.Close() }()

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	bigMeta := map[string]string{"data": strings.Repeat("x", 1024)}

	for i := 0; i < 12000; i++ {
		svc.Log(req, "user", "BULK", "", "ok", bigMeta)
	}

	_ = svc.Close()

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
	defer func() { _ = svc.Close() }()

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
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

func TestAudit_Log_IPExtraction_NoTrustedProxy(t *testing.T) {
	runAuditIPExtractionCase(t, "", []auditIPCase{
		{
			name:       "spoofed XFF is ignored",
			remoteAddr: "192.168.1.1:1234",
			xff:        "1.2.3.4",
			want:       "192.168.1.1",
		},
		{
			name: "empty peer remains valid",
			xff:  "1.2.3.4",
			want: "",
		},
	})
}

func TestAudit_Log_IPExtraction_TrustedProxy(t *testing.T) {
	runAuditIPExtractionCase(t, "127.0.0.1/32", []auditIPCase{
		{
			name:       "first forwarded IP is recorded",
			remoteAddr: "127.0.0.1:1234",
			xff:        " 1.2.3.4, 172.16.0.1",
			want:       "1.2.3.4",
		},
	})
}

func TestAudit_Log_IPExtraction_UntrustedProxy(t *testing.T) {
	runAuditIPExtractionCase(t, "10.0.0.0/8", []auditIPCase{
		{
			name:       "spoofed XFF is ignored",
			remoteAddr: "192.168.1.1:1234",
			xff:        "1.2.3.4",
			want:       "192.168.1.1",
		},
	})
}

type auditIPCase struct {
	name       string
	remoteAddr string
	xff        string
	want       string
}

const auditIPChildEnv = "NRCC_AUDIT_IP_TEST_CHILD"

func runAuditIPExtractionCase(t *testing.T, trustedProxies string, cases []auditIPCase) {
	t.Helper()

	if os.Getenv(auditIPChildEnv) != t.Name() {
		// #nosec G702 -- test fixture: re-invokes the test binary with a fixed -test.run flag; the command is its own OS path, not user input.
		// #nosec G204 -- test fixture: re-invokes the test binary to spawn a child with explicit env.
		cmd := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(t.Name())+"$") //nolint:gosec // G702 G204 test fixtures
		cmd.Env = append(
			environmentWithout("NRCC_TRUSTED_PROXIES", auditIPChildEnv),
			"NRCC_TRUSTED_PROXIES="+trustedProxies,
			auditIPChildEnv+"="+t.Name(),
		)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("child test failed: %v\n%s", err, output)
		}
		return
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, err := NewService(t.TempDir())
			if err != nil {
				t.Fatalf("NewService: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			req.RemoteAddr = tc.remoteAddr
			req.Header.Set("X-Forwarded-For", tc.xff)
			svc.Log(req, "user", "TEST", "", "ok", nil)
			if err := svc.Close(); err != nil {
				t.Fatalf("close service: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(svc.dir, fileName))
			if err != nil {
				t.Fatalf("read log: %v", err)
			}
			var event Event
			if err := json.Unmarshal(data, &event); err != nil {
				t.Fatalf("unmarshal event: %v", err)
			}
			if event.IP != tc.want {
				t.Errorf("IP = %q, want %q", event.IP, tc.want)
			}
		})
	}
}

func environmentWithout(keys ...string) []string {
	excluded := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		excluded[key] = struct{}{}
	}

	env := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, ok := excluded[key]; !ok {
			env = append(env, entry)
		}
	}
	return env
}

func TestLog_SecretsNeverLogged(t *testing.T) {
	svc, _ := NewService(t.TempDir())
	defer func() { _ = svc.Close() }()

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	svc.Log(req, "admin", "ENV_SET", "DB_PASS", "ok", map[string]string{
		"key":  "DB_PASS",
		"type": "secret",
	})
	_ = svc.Close()

	data, _ := os.ReadFile(filepath.Join(svc.dir, fileName))
	raw := string(data)

	if strings.Contains(raw, "password") || strings.Contains(raw, "secret-value") {
		t.Error("audit log should never contain secret values")
	}
}

// TestLog_DocumentsTrustedProxyBoundary verifies REQ-6: the source file
// explicitly documents that audit IP extraction honors NRCC_TRUSTED_PROXIES.
// This is a meta-test against the audit.go source content; it guards against
// silent removal of the trusted-proxy note during future refactors.
func TestLog_DocumentsTrustedProxyBoundary(t *testing.T) {
	data, err := os.ReadFile("audit.go")
	if err != nil {
		t.Fatalf("read audit.go: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, "NRCC_TRUSTED_PROXIES") {
		t.Error("audit.go should document that IP extraction honors NRCC_TRUSTED_PROXIES")
	}
	if !strings.Contains(src, "middleware.ExtractIP") {
		t.Error("audit.go should call middleware.ExtractIP (not a local extractIP)")
	}
	if strings.Contains(src, "func extractIP(") {
		t.Error("audit.go should not contain a local extractIP function (regression of #576)")
	}
}
