package service

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"nrcc/internal/db"
	"nrcc/internal/model"
)

func setupTestJobsService(t *testing.T) (*JobsService, *sql.DB) {
	testDB, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("failed to create in-memory database: %v", err)
	}

	jobsService := NewJobsService(testDB)

	t.Cleanup(func() {
		testDB.Close()
	})

	return jobsService, testDB
}

func insertRawJobRecord(t *testing.T, testDB *sql.DB, id, startedAt string, finishedAt *string) {
	t.Helper()

	createdAt := time.Now().UTC().Format(time.RFC3339)
	_, err := testDB.Exec(
		`INSERT INTO job_history (id, type, status, started_at, finished_at, triggered_by, summary, error, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id,
		model.JobTypeBackup,
		model.JobStatusCompleted,
		startedAt,
		finishedAt,
		"user_123",
		"summary",
		"",
		createdAt,
	)
	if err != nil {
		t.Fatalf("insert raw job record: %v", err)
	}
}

func TestJobsServiceStart(t *testing.T) {
	t.Parallel()

	jobsService, _ := setupTestJobsService(t)

	jobID, err := jobsService.Start(model.JobTypeBackup, "user_123", "Backup database")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if jobID == "" {
		t.Fatal("expected non-empty jobID")
	}

	if !startsWith(jobID, "job_") {
		t.Errorf("expected jobID to start with 'job_', got %s", jobID)
	}

	// Verify job was created in database
	job, err := jobsService.GetByID(jobID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if job.ID != jobID {
		t.Errorf("expected ID %s, got %s", jobID, job.ID)
	}
	if job.Type != model.JobTypeBackup {
		t.Errorf("expected type %s, got %s", model.JobTypeBackup, job.Type)
	}
	if job.Status != model.JobStatusRunning {
		t.Errorf("expected status %s, got %s", model.JobStatusRunning, job.Status)
	}
	if job.TriggeredBy != "user_123" {
		t.Errorf("expected triggeredBy user_123, got %s", job.TriggeredBy)
	}
	if job.Summary != "Backup database" {
		t.Errorf("expected summary 'Backup database', got %s", job.Summary)
	}
}

func TestJobsServiceFinishCompleted(t *testing.T) {
	t.Parallel()

	jobsService, _ := setupTestJobsService(t)

	jobID, _ := jobsService.Start(model.JobTypeBackup, "user_123", "Backup")

	// Finish the job
	err := jobsService.Finish(jobID, model.JobStatusCompleted, "Backup completed successfully", "")
	if err != nil {
		t.Fatalf("Finish() error = %v", err)
	}

	// Verify job status was updated
	job, err := jobsService.GetByID(jobID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if job.Status != model.JobStatusCompleted {
		t.Errorf("expected status %s, got %s", model.JobStatusCompleted, job.Status)
	}
	if job.Summary != "Backup completed successfully" {
		t.Errorf("expected summary, got %s", job.Summary)
	}
	if job.FinishedAt == nil {
		t.Fatal("expected FinishedAt to be set")
	}
}

func TestJobsServiceFinishFailed(t *testing.T) {
	t.Parallel()

	jobsService, _ := setupTestJobsService(t)

	jobID, _ := jobsService.Start(model.JobTypeBackup, "user_123", "Backup")

	// Finish with failure
	err := jobsService.Finish(jobID, model.JobStatusFailed, "", "Backup failed: disk full")
	if err != nil {
		t.Fatalf("Finish() error = %v", err)
	}

	job, err := jobsService.GetByID(jobID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if job.Status != model.JobStatusFailed {
		t.Errorf("expected status %s, got %s", model.JobStatusFailed, job.Status)
	}
	if job.Error != "Backup failed: disk full" {
		t.Errorf("expected error message, got %s", job.Error)
	}
}

func TestJobsServiceGetByID(t *testing.T) {
	t.Parallel()

	jobsService, _ := setupTestJobsService(t)

	jobID, _ := jobsService.Start(model.JobTypeNpmInstall, "user_456", "Install package")

	job, err := jobsService.GetByID(jobID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if job == nil {
		t.Fatal("expected job to be returned, got nil")
	}
	if job.ID != jobID {
		t.Errorf("expected ID %s, got %s", jobID, job.ID)
	}
}

func TestJobsServiceGetByIDNotFound(t *testing.T) {
	t.Parallel()

	jobsService, _ := setupTestJobsService(t)

	_, err := jobsService.GetByID("job_nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent job")
	}
}

func TestJobsServiceGetByIDReturnsParseErrorForInvalidStartedAt(t *testing.T) {
	t.Parallel()

	jobsService, testDB := setupTestJobsService(t)
	insertRawJobRecord(t, testDB, "job_invalid_started", "not-a-timestamp", nil)

	_, err := jobsService.GetByID("job_invalid_started")
	if err == nil {
		t.Fatal("expected parse error for invalid started_at")
	}
	if !strings.Contains(err.Error(), "parse started_at") {
		t.Fatalf("expected started_at parse error, got %v", err)
	}
}

func TestJobsServiceGetWithTypeFilter(t *testing.T) {
	t.Parallel()

	jobsService, _ := setupTestJobsService(t)

	// Create jobs of different types
	jobsService.Start(model.JobTypeBackup, "user_1", "Backup 1")
	jobsService.Start(model.JobTypeBackup, "user_1", "Backup 2")
	jobsService.Start(model.JobTypeNpmInstall, "user_1", "Install")
	jobsService.Start(model.JobTypeRestart, "user_1", "Restart")

	// Get only backup jobs
	jobs, err := jobsService.Get(50, 0, model.JobTypeBackup, "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(jobs) != 2 {
		t.Fatalf("expected 2 backup jobs, got %d", len(jobs))
	}

	for _, job := range jobs {
		if job.Type != model.JobTypeBackup {
			t.Errorf("expected type backup, got %s", job.Type)
		}
	}
}

func TestJobsServiceGetReturnsParseErrorForInvalidFinishedAt(t *testing.T) {
	t.Parallel()

	jobsService, testDB := setupTestJobsService(t)
	finishedAt := "not-a-timestamp"
	insertRawJobRecord(t, testDB, "job_invalid_finished", time.Now().UTC().Format(time.RFC3339), &finishedAt)

	_, err := jobsService.Get(50, 0, "", "")
	if err == nil {
		t.Fatal("expected parse error for invalid finished_at")
	}
	if !strings.Contains(err.Error(), "parse finished_at") {
		t.Fatalf("expected finished_at parse error, got %v", err)
	}
}

func TestJobsServiceGetWithStatusFilter(t *testing.T) {
	t.Parallel()

	jobsService, _ := setupTestJobsService(t)

	// Create jobs with different statuses
	jobID1, _ := jobsService.Start(model.JobTypeBackup, "user_1", "Backup 1")
	jobID2, _ := jobsService.Start(model.JobTypeBackup, "user_1", "Backup 2")
	jobID3, _ := jobsService.Start(model.JobTypeBackup, "user_1", "Backup 3")

	// Complete one job
	jobsService.Finish(jobID1, model.JobStatusCompleted, "Success", "")

	// Fail one job
	jobsService.Finish(jobID2, model.JobStatusFailed, "", "Error")

	// Leave one running

	// Get completed jobs
	jobs, err := jobsService.Get(50, 0, "", model.JobStatusCompleted)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 completed job, got %d", len(jobs))
	}
	if jobs[0].ID != jobID1 {
		t.Errorf("expected job %s, got %s", jobID1, jobs[0].ID)
	}

	// Get failed jobs
	jobs, err = jobsService.Get(50, 0, "", model.JobStatusFailed)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 failed job, got %d", len(jobs))
	}
	if jobs[0].ID != jobID2 {
		t.Errorf("expected job %s, got %s", jobID2, jobs[0].ID)
	}

	// Get running jobs
	jobs, err = jobsService.Get(50, 0, "", model.JobStatusRunning)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("expected 1 running job, got %d", len(jobs))
	}
	if jobs[0].ID != jobID3 {
		t.Errorf("expected job %s, got %s", jobID3, jobs[0].ID)
	}
}

func TestJobsServiceGetWithLimit(t *testing.T) {
	t.Parallel()

	jobsService, _ := setupTestJobsService(t)

	// Create 10 jobs
	for i := 0; i < 10; i++ {
		jobsService.Start(model.JobTypeBackup, "user", "Backup")
	}

	// Get with limit 5
	jobs, err := jobsService.Get(5, 0, "", "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(jobs) != 5 {
		t.Fatalf("expected 5 jobs with limit 5, got %d", len(jobs))
	}
}

func TestJobsServiceGetWithOffset(t *testing.T) {
	t.Parallel()

	jobsService, _ := setupTestJobsService(t)

	// Create 10 jobs
	for i := 0; i < 10; i++ {
		jobsService.Start(model.JobTypeBackup, "user", "Backup")
	}

	// Get first 3 jobs
	jobs1, err := jobsService.Get(3, 0, "", "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(jobs1) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs1))
	}

	// Get next 3 jobs with offset
	jobs2, err := jobsService.Get(3, 3, "", "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(jobs2) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs2))
	}

	// Ensure different jobs
	if jobs1[0].ID == jobs2[0].ID {
		t.Fatal("expected different jobs between offset results")
	}
}

func TestJobsServiceGetDefaultLimit(t *testing.T) {
	t.Parallel()

	jobsService, _ := setupTestJobsService(t)

	// Create 100 jobs (more than default limit of 50)
	for i := 0; i < 100; i++ {
		jobsService.Start(model.JobTypeBackup, "user", "Backup")
	}

	// Get with limit 0 (should use default 50)
	jobs, err := jobsService.Get(0, 0, "", "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(jobs) != 50 {
		t.Fatalf("expected default limit 50, got %d", len(jobs))
	}
}

func TestJobsServiceGetOrderByNewest(t *testing.T) {
	t.Parallel()

	jobsService, _ := setupTestJobsService(t)

	// Create 3 jobs with small delays
	jobID1, _ := jobsService.Start(model.JobTypeBackup, "user", "Backup 1")
	time.Sleep(10 * time.Millisecond)
	jobID2, _ := jobsService.Start(model.JobTypeBackup, "user", "Backup 2")
	time.Sleep(10 * time.Millisecond)
	jobID3, _ := jobsService.Start(model.JobTypeBackup, "user", "Backup 3")

	jobs, err := jobsService.Get(50, 0, "", "")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Should be in reverse chronological order (newest first)
	if jobs[0].ID != jobID3 {
		t.Errorf("expected first job to be %s, got %s", jobID3, jobs[0].ID)
	}
	if jobs[1].ID != jobID2 {
		t.Errorf("expected second job to be %s, got %s", jobID2, jobs[1].ID)
	}
	if jobs[2].ID != jobID1 {
		t.Errorf("expected third job to be %s, got %s", jobID1, jobs[2].ID)
	}
}

func TestJobContextStartStop(t *testing.T) {
	t.Parallel()

	jobsService, db := setupTestJobsService(t)

	logService, err := NewLogService(t.TempDir(), db)
	if err != nil {
		t.Fatalf("NewLogService() error = %v", err)
	}
	defer logService.Close()

	// Create a job context
	jc, err := NewJobContext(jobsService, logService, model.JobTypeBackup, "user_123", "Backup data")
	if err != nil {
		t.Fatalf("NewJobContext() error = %v", err)
	}

	if jc.JobID == "" {
		t.Fatal("expected non-empty JobID")
	}

	// Complete the job
	err = jc.Complete("Backup finished successfully")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	// Verify job status
	job, err := jobsService.GetByID(jc.JobID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if job.Status != model.JobStatusCompleted {
		t.Errorf("expected status %s, got %s", model.JobStatusCompleted, job.Status)
	}
}

func TestJobContextFail(t *testing.T) {
	t.Parallel()

	jobsService, db := setupTestJobsService(t)

	logService, _ := NewLogService(t.TempDir(), db)
	defer logService.Close()

	jc, _ := NewJobContext(jobsService, logService, model.JobTypeBackup, "user_123", "Backup data")

	// Fail the job
	err := jc.Fail("Backup failed: disk full")
	if err != nil {
		t.Fatalf("Fail() error = %v", err)
	}

	// Verify job status
	job, err := jobsService.GetByID(jc.JobID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if job.Status != model.JobStatusFailed {
		t.Errorf("expected status %s, got %s", model.JobStatusFailed, job.Status)
	}
	if job.Error != "Backup failed: disk full" {
		t.Errorf("expected error message, got %s", job.Error)
	}
}

func TestJobContextLog(t *testing.T) {
	t.Parallel()

	jobsService, db := setupTestJobsService(t)

	logService, _ := NewLogService(t.TempDir(), db)
	defer logService.Close()

	jc, _ := NewJobContext(jobsService, logService, model.JobTypeBackup, "user_123", "Backup")

	// Log a message
	err := jc.Log(model.EventJobStarted, "Starting backup process", model.LogLevelInfo)
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	// Verify log was created
	logs := logService.Get(100, "", "")
	// Should have at least 2 logs (job started + our custom log)
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 logs, got %d", len(logs))
	}
}
