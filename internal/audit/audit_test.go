package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
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
	wrapOpen     func(auditFile) auditFile
	linkErr      error
	openErr      error
	openFailures int
	removeErr    map[string]error
	statErr      map[string]error
}

func (f *testFS) OpenFile(name string, flag int, perm os.FileMode) (auditFile, error) {
	if f.openErr != nil && (f.openFailures < 0 || f.openFailures > 0) {
		if f.openFailures > 0 {
			f.openFailures--
		}
		return nil, f.openErr
	}
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

func (f *testFS) Remove(path string) error {
	if err := f.removeErr[filepath.Base(path)]; err != nil {
		return err
	}
	return f.osFileSystem.Remove(path)
}

func (f *testFS) Stat(path string) (os.FileInfo, error) {
	if err := f.statErr[filepath.Base(path)]; err != nil {
		return nil, err
	}
	return f.osFileSystem.Stat(path)
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

func TestService_RotationRecovery(t *testing.T) {
	t.Run("OS hard links preserve same-clock payloads", func(t *testing.T) {
		now := time.Date(2026, time.August, 15, 12, 34, 56, 123456789, time.UTC)
		svc, err := newService(t.TempDir(), testDependencies(now, &testFS{}, &reportRecorder{}))
		if err != nil {
			t.Fatalf("newService: %v", err)
		}
		defer func() { _ = svc.Close() }()
		for _, action := range []string{"FIRST", "SECOND"} {
			svc.size = maxSize
			outcome := svc.Log(httptest.NewRequest(http.MethodPost, "/", nil), "a", action, "", "ok", nil)
			if !outcome.Persisted || outcome.MaintenanceErr != nil {
				t.Fatalf("Log(%s) = %+v", action, outcome)
			}
		}
		entries, err := os.ReadDir(svc.dir)
		if err != nil {
			t.Fatal(err)
		}
		var payloads []string
		for _, entry := range entries {
			if isBackupName(entry.Name()) {
				data, err := os.ReadFile(filepath.Join(svc.dir, entry.Name()))
				if err != nil {
					t.Fatal(err)
				}
				payloads = append(payloads, string(data))
			}
		}
		if len(payloads) != 2 || !strings.Contains(payloads[0], `"action":"FIRST"`) || !strings.Contains(payloads[1], `"action":"SECOND"`) {
			t.Fatalf("rotated payloads = %q, want FIRST then SECOND", payloads)
		}
	})

	t.Run("link failure recovers truthful size", func(t *testing.T) {
		fs := &testFS{linkErr: errors.New("link denied")}
		svc, err := newService(t.TempDir(), testDependencies(time.Now(), fs, &reportRecorder{}))
		if err != nil {
			t.Fatalf("newService: %v", err)
		}
		svc.size = maxSize
		outcome := svc.Log(httptest.NewRequest(http.MethodPost, "/", nil), "a", "A", "", "ok", nil)
		if !outcome.Persisted || outcome.MaintenanceErr == nil || svc.file == nil {
			t.Fatalf("outcome=%+v file=%v, want persisted maintenance failure and recovered file", outcome, svc.file)
		}
		info, err := os.Stat(filepath.Join(svc.dir, fileName))
		if err != nil || svc.size != info.Size() {
			t.Fatalf("size=%d stat=(%v,%v), want truthful recovered size", svc.size, info, err)
		}
		fs.linkErr = nil
		later := svc.Log(httptest.NewRequest(http.MethodPost, "/", nil), "b", "B", "", "ok", nil)
		if !later.Persisted || later.EventErr != nil || later.MaintenanceErr != nil {
			t.Fatalf("later Log = %+v, want successful persistence", later)
		}
	})

	t.Run("replacement open failure recovers", func(t *testing.T) {
		fs := &testFS{}
		svc, err := newService(t.TempDir(), testDependencies(time.Now(), fs, &reportRecorder{}))
		if err != nil {
			t.Fatalf("newService: %v", err)
		}
		fs.openErr, fs.openFailures = errors.New("open interrupted"), 1
		svc.size = maxSize
		outcome := svc.Log(httptest.NewRequest(http.MethodPost, "/", nil), "a", "A", "", "ok", nil)
		if !outcome.Persisted || outcome.MaintenanceErr == nil || outcome.MaintenanceErr.Stage != "open-replacement" || svc.file == nil || svc.size != 0 {
			t.Fatalf("outcome=%+v file=%v size=%d, want recovered empty active", outcome, svc.file, svc.size)
		}
		later := svc.Log(httptest.NewRequest(http.MethodPost, "/", nil), "b", "B", "", "ok", nil)
		if !later.Persisted || later.EventErr != nil || later.MaintenanceErr != nil {
			t.Fatalf("later Log = %+v, want successful persistence", later)
		}
	})

	t.Run("active remove rollback and pending alias retry", func(t *testing.T) {
		now := time.Date(2026, time.August, 15, 12, 34, 56, 0, time.UTC)
		candidate := "audit-20260815-123456.000000000-000000.jsonl"
		for _, tc := range []struct {
			name            string
			failRollback    bool
			wantPendingPath bool
		}{
			{name: "rollback recovers active writer"},
			{name: "pending alias is removed before retry", failRollback: true, wantPendingPath: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				fs := &testFS{removeErr: map[string]error{fileName: errors.New("active remove denied")}}
				if tc.failRollback {
					fs.removeErr[candidate] = errors.New("rollback remove denied")
				}
				svc, err := newService(t.TempDir(), testDependencies(now, fs, &reportRecorder{}))
				if err != nil {
					t.Fatal(err)
				}
				svc.size = maxSize
				first := svc.Log(httptest.NewRequest(http.MethodPost, "/", nil), "a", "A", "", "ok", nil)
				if !first.Persisted || first.MaintenanceErr == nil || first.MaintenanceErr.Stage != "remove-active" {
					t.Fatalf("first Log = %+v, want persisted remove-active failure", first)
				}
				if got := svc.pendingBackup != ""; got != tc.wantPendingPath {
					t.Fatalf("pendingBackup set = %v, want %v", got, tc.wantPendingPath)
				}
				fs.removeErr = nil
				later := svc.Log(httptest.NewRequest(http.MethodPost, "/", nil), "b", "B", "", "ok", nil)
				if !later.Persisted || later.EventErr != nil || later.MaintenanceErr != nil {
					t.Fatalf("later Log = %+v, want successful persistence", later)
				}
				if _, err := os.Stat(filepath.Join(svc.dir, candidate)); tc.wantPendingPath && !os.IsNotExist(err) {
					t.Fatalf("pending alias still exists: %v", err)
				}
			})
		}
	})

	t.Run("unrecoverable writer rejects later event", func(t *testing.T) {
		fs := &testFS{}
		svc, err := newService(t.TempDir(), testDependencies(time.Now(), fs, &reportRecorder{}))
		if err != nil {
			t.Fatalf("newService: %v", err)
		}
		fs.openErr, fs.openFailures = errors.New("open denied"), -1
		svc.size = maxSize
		first := svc.Log(httptest.NewRequest(http.MethodPost, "/", nil), "a", "A", "", "ok", nil)
		second := svc.Log(httptest.NewRequest(http.MethodPost, "/", nil), "b", "B", "", "ok", nil)
		if !first.Persisted || first.MaintenanceErr == nil || svc.file != nil || svc.size != 0 {
			t.Fatalf("first=%+v file=%v size=%d", first, svc.file, svc.size)
		}
		if second.Persisted || second.EventErr == nil || second.EventErr.Stage != "unavailable-writer" {
			t.Fatalf("second=%+v, want unavailable writer", second)
		}
	})

	t.Run("startup removes uncommitted hard-link alias", func(t *testing.T) {
		dataDir := t.TempDir()
		auditDir := filepath.Join(dataDir, "audit")
		if err := os.MkdirAll(auditDir, 0700); err != nil {
			t.Fatal(err)
		}
		active := filepath.Join(auditDir, fileName)
		alias := filepath.Join(auditDir, "audit-20260815-123456.000000000-000000.jsonl")
		if err := os.WriteFile(active, []byte("existing\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(active, alias); err != nil {
			t.Fatal(err)
		}
		svc, err := newService(dataDir, testDependencies(time.Now(), &testFS{}, &reportRecorder{}))
		if err != nil {
			t.Fatalf("newService: %v", err)
		}
		defer func() { _ = svc.Close() }()
		if _, err := os.Stat(alias); !os.IsNotExist(err) {
			t.Fatalf("alias still exists: %v", err)
		}
		if svc.size != int64(len("existing\n")) {
			t.Fatalf("size=%d, want %d", svc.size, len("existing\n"))
		}
	})
}

func TestService_Prune(t *testing.T) {
	now := time.Date(2026, time.August, 15, 12, 34, 56, 0, time.UTC)
	t.Run("same timestamp ordinals retain newest", func(t *testing.T) {
		svc, err := newService(t.TempDir(), testDependencies(now, &testFS{}, &reportRecorder{}))
		if err != nil {
			t.Fatal(err)
		}
		for ordinal := 0; ordinal < maxBackups+2; ordinal++ {
			name := fmt.Sprintf("audit-20260815-123456.000000000-%06d.jsonl", ordinal)
			if err := os.WriteFile(filepath.Join(svc.dir, name), []byte(name), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if opErr := svc.pruneOld(); opErr != nil {
			t.Fatalf("pruneOld: %v", opErr)
		}
		for ordinal := 0; ordinal < maxBackups+2; ordinal++ {
			name := fmt.Sprintf("audit-20260815-123456.000000000-%06d.jsonl", ordinal)
			_, err := os.Stat(filepath.Join(svc.dir, name))
			if ordinal < 2 && !os.IsNotExist(err) {
				t.Fatalf("old ordinal %d retained: %v", ordinal, err)
			}
			if ordinal >= 2 && err != nil {
				t.Fatalf("new ordinal %d missing: %v", ordinal, err)
			}
		}
	})

	t.Run("delete failure after persistence keeps writer usable", func(t *testing.T) {
		fs := &testFS{}
		reporter := &reportRecorder{}
		svc, err := newService(t.TempDir(), testDependencies(now, fs, reporter))
		if err != nil {
			t.Fatal(err)
		}
		oldest := "audit-20260815-123450.000000000-000000.jsonl"
		for i := 0; i < maxBackups; i++ {
			name := fmt.Sprintf("audit-20260815-12345%d.000000000-000000.jsonl", i)
			if err := os.WriteFile(filepath.Join(svc.dir, name), []byte(name), 0600); err != nil {
				t.Fatal(err)
			}
		}
		fs.removeErr = map[string]error{oldest: errors.New("remove denied")}
		svc.size = maxSize
		outcome := svc.Log(httptest.NewRequest(http.MethodPost, "/", nil), "a", "A", "", "ok", nil)
		if !outcome.Persisted || outcome.MaintenanceErr == nil || outcome.MaintenanceErr.Stage != "prune-backup" {
			t.Fatalf("Log = %+v, want persisted prune failure", outcome)
		}
		if len(reporter.reports) != 1 || !reporter.reports[0].Persisted || reporter.reports[0].Stage != "prune-backup" {
			t.Fatalf("reports = %+v, want persisted prune maintenance report", reporter.reports)
		}
		fs.removeErr = nil
		later := svc.Log(httptest.NewRequest(http.MethodPost, "/", nil), "b", "B", "", "ok", nil)
		if !later.Persisted || later.EventErr != nil || later.MaintenanceErr != nil {
			t.Fatalf("later Log = %+v, want successful persistence", later)
		}
	})
	t.Run("mixed names retain newest rotation keys", func(t *testing.T) {
		dataDir := t.TempDir()
		fs := &testFS{}
		svc, err := newService(dataDir, testDependencies(now, fs, &reportRecorder{}))
		if err != nil {
			t.Fatal(err)
		}
		names := []string{
			"audit-20260815-123455.jsonl",
			"audit-20260815-123456.100000000-000000.jsonl",
			"audit-20260815-123456.jsonl",
			"audit-20260815-123456.300000000-000000.jsonl",
			"audit-malformed-a.jsonl",
			"audit-malformed-b.jsonl",
			"audit-20260815-123457.000000000-000000.jsonl",
		}
		for i, name := range names {
			path := filepath.Join(svc.dir, name)
			if err := os.WriteFile(path, []byte(name), 0600); err != nil {
				t.Fatal(err)
			}
			mtime := now.Add(time.Duration(i-2) * 100 * time.Millisecond)
			if err := os.Chtimes(path, mtime, mtime); err != nil {
				t.Fatal(err)
			}
		}
		if err := svc.pruneOld(); err != nil {
			t.Fatalf("pruneOld: %v", err)
		}
		entries, _ := os.ReadDir(svc.dir)
		var backups []string
		for _, entry := range entries {
			if entry.Name() != fileName {
				backups = append(backups, entry.Name())
			}
		}
		if len(backups) != maxBackups {
			t.Fatalf("backups=%v, want %d retained", backups, maxBackups)
		}
		if slices.Contains(backups, names[0]) || slices.Contains(backups, names[2]) {
			t.Fatalf("oldest backups retained: %v", backups)
		}
	})

	for _, tc := range []struct {
		name      string
		configure func(*testFS, string)
		wantStage string
	}{
		{"info failure aborts deletion", func(fs *testFS, name string) { fs.statErr = map[string]error{name: errors.New("stat denied")} }, "inspect-backup"},
		{"delete failure is reported", func(fs *testFS, name string) { fs.removeErr = map[string]error{name: errors.New("remove denied")} }, "prune-backup"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := &testFS{}
			svc, err := newService(t.TempDir(), testDependencies(now, fs, &reportRecorder{}))
			if err != nil {
				t.Fatal(err)
			}
			oldest := "audit-20260815-123450.000000000-000000.jsonl"
			for i := 0; i <= maxBackups; i++ {
				name := fmt.Sprintf("audit-20260815-12345%d.000000000-000000.jsonl", i)
				if err := os.WriteFile(filepath.Join(svc.dir, name), []byte(name), 0600); err != nil {
					t.Fatal(err)
				}
			}
			tc.configure(fs, oldest)
			opErr := svc.pruneOld()
			if opErr == nil || opErr.Stage != tc.wantStage {
				t.Fatalf("prune error=%+v, want stage %q", opErr, tc.wantStage)
			}
			if _, err := os.Stat(filepath.Join(svc.dir, oldest)); err != nil {
				t.Fatalf("oldest removed despite failure: %v", err)
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
	outcome := svc.Log(req, "x", "X", "", "ok", nil)
	if outcome.Persisted || outcome.EventErr == nil || outcome.EventErr.Stage != "unavailable-writer" {
		t.Fatalf("Log = %+v, want unavailable writer outcome", outcome)
	}
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
