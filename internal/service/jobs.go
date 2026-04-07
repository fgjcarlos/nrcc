package service

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"nrcc/internal/model"
)

// JobsService manages job history in SQLite
type JobsService struct {
	db *sql.DB
}

// NewJobsService creates a new JobsService
func NewJobsService(db *sql.DB) *JobsService {
	return &JobsService{
		db: db,
	}
}

// Start creates a new job record and returns its ID
func (js *JobsService) Start(jobType, triggeredBy, summary string) (string, error) {
	jobID := "job_" + uuid.New().String()
	startedAt := time.Now().UTC()

	query := `
		INSERT INTO job_history (id, type, status, started_at, triggered_by, summary, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := js.db.Exec(query, jobID, jobType, model.JobStatusRunning, startedAt, triggeredBy, summary, startedAt)
	if err != nil {
		return "", fmt.Errorf("insert job record: %w", err)
	}

	return jobID, nil
}

// Finish updates a job record with completion status
func (js *JobsService) Finish(jobID, status, summary, errMsg string) error {
	finishedAt := time.Now().UTC()

	query := `
		UPDATE job_history
		SET status = ?, finished_at = ?, summary = ?, error = ?
		WHERE id = ?
	`
	_, err := js.db.Exec(query, status, finishedAt, summary, errMsg, jobID)
	if err != nil {
		return fmt.Errorf("update job record: %w", err)
	}

	return nil
}

// Get retrieves jobs with pagination and optional filters
func (js *JobsService) Get(limit, offset int, jobType, status string) ([]model.JobRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, type, status, started_at, finished_at, triggered_by, summary, error
		FROM job_history
		WHERE 1=1
	`
	args := []any{}

	if jobType != "" {
		query += ` AND type = ?`
		args = append(args, jobType)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}

	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := js.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []model.JobRecord
	for rows.Next() {
		var job model.JobRecord
		var finishedAtStr sql.NullString

		err := rows.Scan(&job.ID, &job.Type, &job.Status, &job.StartedAt, &finishedAtStr, &job.TriggeredBy, &job.Summary, &job.Error)
		if err != nil {
			return nil, fmt.Errorf("scan job row: %w", err)
		}

		if finishedAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, finishedAtStr.String)
			job.FinishedAt = &t
		}

		jobs = append(jobs, job)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate job rows: %w", err)
	}

	return jobs, nil
}

// GetByID retrieves a specific job by ID
func (js *JobsService) GetByID(jobID string) (*model.JobRecord, error) {
	query := `
		SELECT id, type, status, started_at, finished_at, triggered_by, summary, error
		FROM job_history
		WHERE id = ?
	`
	var job model.JobRecord
	var finishedAtStr sql.NullString

	err := js.db.QueryRow(query, jobID).Scan(
		&job.ID, &job.Type, &job.Status, &job.StartedAt, &finishedAtStr, &job.TriggeredBy, &job.Summary, &job.Error,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, fmt.Errorf("query job: %w", err)
	}

	if finishedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, finishedAtStr.String)
		job.FinishedAt = &t
	}

	return &job, nil
}
