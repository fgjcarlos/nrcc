package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/audit"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
)

const concurrentApplyRequests = 32

// A burst of POST /apply requests from any startable state must run the update
// flow exactly once. The run is claimed before the response, so exactly one
// request gets 200 (and one audit entry); every other request gets 409 and
// leaves state and files untouched.
func TestUpdateHandler_PostApply_ConcurrentRequestsRunOneFlow(t *testing.T) {
	for _, start := range []model.UpdateState{model.StateIdle, model.StateCompleted, model.StateFailed} {
		t.Run(string(start), func(t *testing.T) {
			installFakeNPM(t)
			dataDir := t.TempDir()
			cachePath := filepath.Join(dataDir, "update_cache.json")
			historyPath := filepath.Join(dataDir, "update_backups.json")
			if err := store.NewJSONStore[model.UpdateCacheEntry](cachePath).Write(model.UpdateCacheEntry{
				CurrentVersion: "4.0.1", LatestVersion: "4.0.2", UpdateAvailable: true, CheckedAt: time.Now().UTC(),
			}); err != nil {
				t.Fatal(err)
			}
			if err := store.NewJSONStore[[]model.BackupEntry](historyPath).Write([]model.BackupEntry{
				{ID: "previous-update", FromVersion: "4.0.0", Status: "completed", Timestamp: time.Now().UTC()},
			}); err != nil {
				t.Fatal(err)
			}

			svc := service.NewUpdateService(dataDir)
			// Proves historyPath is the file the service reads as update history.
			if h := svc.History(); len(h) != 1 || h[0].ID != "previous-update" {
				t.Fatalf("seeded history not visible through the service: %+v", h)
			}
			// Buffered so every backup attempt is recorded without blocking on the test.
			creator := gatedBackupCreator{started: make(chan struct{}, concurrentApplyRequests), release: make(chan struct{})}
			svc.SetBackupCreator(creator)
			svc.SetFlowState(model.UpdateFlowState{State: start, Phase: "done"})
			handler := NewUpdateHandler(svc)
			auditDir := t.TempDir()
			auditSvc, err := audit.NewService(auditDir)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = auditSvc.Close() })
			handler.SetAuditService(auditSvc)

			cacheBefore, historyBefore := fileSHA256(t, cachePath), fileSHA256(t, historyPath)

			codes := make([]int, concurrentApplyRequests)
			fire := make(chan struct{})
			var wg sync.WaitGroup
			for i := range codes {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-fire
					w := httptest.NewRecorder()
					handler.PostApply(w, httptest.NewRequest(http.MethodPost, "/api/updates/apply", nil))
					codes[i] = w.Code
				}()
			}
			close(fire)
			wg.Wait()

			// Wait until every launched flow goroutine except the one holding the
			// gate has returned, so none can start a second run after release.
			deadline := time.Now().Add(5 * time.Second)
			for len(creator.started) == 0 || runningApplyFlows() > 1 {
				if time.Now().After(deadline) {
					t.Fatalf("flows did not settle on one gated run: backups=%d running=%d state=%+v",
						len(creator.started), runningApplyFlows(), svc.GetFlowState())
				}
				time.Sleep(5 * time.Millisecond)
			}

			close(creator.release)
			deadline = time.Now().Add(5 * time.Second)
			for svc.GetFlowState().State != model.StateFailed || runningApplyFlows() > 0 {
				if time.Now().After(deadline) {
					t.Fatalf("released update did not settle; state=%+v", svc.GetFlowState())
				}
				time.Sleep(5 * time.Millisecond)
			}

			if n := len(creator.started); n != 1 {
				t.Fatalf("backup step ran %d times, want exactly 1", n)
			}
			if got := svc.GetFlowState(); got.State != model.StateFailed || got.Error != "backup_failed" {
				t.Fatalf("final flow state = %+v, want Failed/backup_failed", got)
			}
			accepted := 0
			for i, code := range codes {
				switch code {
				case http.StatusOK:
					accepted++
				case http.StatusConflict:
				default:
					t.Errorf("request %d returned %d, want 200 or 409", i, code)
				}
			}
			if accepted != 1 {
				t.Errorf("%d requests were accepted, want exactly 1; codes=%v", accepted, codes)
			}
			if n := countAuditActions(t, auditDir, "UPDATE_APPLY"); n != 1 {
				t.Errorf("UPDATE_APPLY audit entries = %d, want exactly 1", n)
			}
			if got := fileSHA256(t, cachePath); got != cacheBefore {
				t.Errorf("update_cache.json changed: %x -> %x", cacheBefore, got)
			}
			if got := fileSHA256(t, historyPath); got != historyBefore {
				t.Errorf("update_backups.json changed: %x -> %x", historyBefore, got)
			}
		})
	}
}

// An accepted request must already own the run when it responds. Two back-to-back
// requests from a terminal state used to both get 200; the second goroutine then
// ran "late", after the first run had finished (Failed is startable again), and
// started a second update. Now the second request is rejected with 409.
func TestUpdateHandler_PostApply_AcceptedRequestClaimsRunBeforeResponding(t *testing.T) {
	// One P keeps launched goroutines from running until the test blocks, which
	// reproduces the late-scheduled goroutine deterministically.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))

	installFakeNPM(t)
	dataDir := t.TempDir()
	// A known latest version and stale fields from the previous run let the
	// state assertions below catch any field the claim must not carry. The
	// cache is up to date: GetFlowState overlays availableVersion only when an
	// update is available, so this is where a claimed value would be visible.
	if err := store.NewJSONStore[model.UpdateCacheEntry](filepath.Join(dataDir, "update_cache.json")).Write(model.UpdateCacheEntry{
		CurrentVersion: "4.0.2", LatestVersion: "4.0.2", UpdateAvailable: false, CheckedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	svc := service.NewUpdateService(dataDir)
	creator := &countingFailingBackupCreator{}
	svc.SetBackupCreator(creator)
	svc.SetFlowState(model.UpdateFlowState{State: model.StateCompleted, Phase: "completed", BackupID: "stale-id", Error: "stale-error"})
	handler := NewUpdateHandler(svc)

	post := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.PostApply(w, httptest.NewRequest(http.MethodPost, "/api/updates/apply", nil))
		return w
	}
	first, second := post(), post()

	if first.Code != http.StatusOK {
		t.Fatalf("first POST /apply = %d, want 200; body: %s", first.Code, first.Body.String())
	}
	var resp model.ApiResponse[map[string]any]
	if err := json.Unmarshal(first.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if got := resp.Data["state"]; got != string(model.StateBackingUp) {
		t.Errorf("accepted response state = %v, want BackingUp", got)
	}
	if v := resp.Data["backupId"]; v != "" {
		t.Errorf("accepted response backupId = %v, want empty", v)
	}
	// GET /state while the run is still claimed: the fresh BackingUp state has
	// no stale backup ID or error, and no field it did not carry before the
	// synchronous claim. The single P has not run the launched goroutine yet.
	stateRec := httptest.NewRecorder()
	handler.GetState(stateRec, httptest.NewRequest(http.MethodGet, "/api/updates/state", nil))
	var inFlight model.ApiResponse[map[string]any]
	if err := json.Unmarshal(stateRec.Body.Bytes(), &inFlight); err != nil {
		t.Fatal(err)
	}
	if got := inFlight.Data["state"]; got != string(model.StateBackingUp) {
		t.Errorf("in-flight GET /state state = %v, want BackingUp; body: %s", got, stateRec.Body.String())
	}
	for _, field := range []string{"backupId", "error", "availableVersion"} {
		if v, ok := inFlight.Data[field]; ok {
			t.Errorf("in-flight GET /state %s = %v, want absent", field, v)
		}
	}
	var conflict model.ApiResponse[any]
	if err := json.Unmarshal(second.Body.Bytes(), &conflict); err != nil {
		t.Fatal(err)
	}
	if second.Code != http.StatusConflict || conflict.Error == nil || conflict.Error.Code != "UPDATE_IN_PROGRESS" {
		t.Errorf("second POST /apply = %d %s, want 409 UPDATE_IN_PROGRESS", second.Code, second.Body.String())
	}

	deadline := time.Now().Add(5 * time.Second)
	for svc.GetFlowState().State != model.StateFailed || runningApplyFlows() > 0 {
		if time.Now().After(deadline) {
			t.Fatalf("update flows did not settle; state=%+v", svc.GetFlowState())
		}
		time.Sleep(5 * time.Millisecond)
	}
	if n := creator.calls.Load(); n != 1 {
		t.Errorf("backup step ran %d times, want exactly 1", n)
	}
}

// countingFailingBackupCreator records each backup attempt and fails it, so a
// run reaches Failed (a startable state) without touching the network.
type countingFailingBackupCreator struct{ calls atomic.Int32 }

func (c *countingFailingBackupCreator) CreateTyped(model.BackupType, string) (model.Backup, error) {
	c.calls.Add(1)
	return model.Backup{}, errors.New("backup refused by test")
}

func countAuditActions(t *testing.T, auditDataDir, action string) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(auditDataDir, "audit", "audit.jsonl")) // #nosec G304 -- path is inside t.TempDir().
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Count(data, []byte(`"action":"`+action+`"`))
}

func fileSHA256(t *testing.T, path string) [sha256.Size]byte {
	t.Helper()
	data, err := os.ReadFile(path) // #nosec G304 -- path is inside t.TempDir().
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(data)
}

// runningApplyFlows counts live goroutines launched by PostApply. Matching the
// closure frame ignores unrelated goroutines from other tests or the runtime.
func runningApplyFlows() int {
	buf := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	count := 0
	for _, g := range bytes.Split(buf, []byte("\n\n")) {
		if bytes.Contains(g, []byte("(*UpdateHandler).PostApply.func1(")) {
			count++
		}
	}
	return count
}
