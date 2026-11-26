package service

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestBackupSchedulerRunsSessionPruningDaily(t *testing.T) {
	dir := t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		backup := NewBackupService(dir)
		var runs atomic.Int32
		backup.ScheduleRefreshSessionPruning(func() error {
			runs.Add(1)
			return nil
		})
		checkRuns := func(want int32) {
			t.Helper()
			synctest.Wait()
			if got := runs.Load(); got != want {
				t.Fatalf("runs at %v = %d, want %d", time.Now(), got, want)
			}
		}
		checkRuns(1)
		backup.Start(ctx)
		checkRuns(1)
		time.Sleep(24*time.Hour - time.Second)
		checkRuns(1)
		time.Sleep(time.Second)
		checkRuns(2)
		time.Sleep(24 * time.Hour)
		checkRuns(3)
		cancel()
		synctest.Wait()
		time.Sleep(24 * time.Hour)
		checkRuns(3)
	})
}
