package handler

import (
	"bytes"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
	"github.com/fgjcarlos/nrcc/internal/store"
)

const concurrentApplyRequests = 32

// A burst of POST /apply requests from any startable state must run the update
// flow exactly once. Requests that pass the handler's state check before the
// flow reaches BackingUp still return 200, but their goroutines must lose
// applyMu.TryLock in the service and leave state and files untouched.
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
			if accepted == 0 {
				t.Errorf("no request was accepted; codes=%v", codes)
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
