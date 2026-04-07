package service

import (
	"fmt"
	"time"

	"nrcc/internal/model"
)

// JobContext provides context for a running job with automatic logging
type JobContext struct {
	JobID       string
	jobsService *JobsService
	logService  *LogService
	jobType     string
	triggeredBy string
}

// NewJobContext creates a new JobContext and starts a job
func NewJobContext(js *JobsService, ls *LogService, jobType, triggeredBy, summary string) (*JobContext, error) {
	jobID, err := js.Start(jobType, triggeredBy, summary)
	if err != nil {
		return nil, err
	}

	// Log job start
	if ls != nil {
		entry := model.LogEntry{
			Level:     model.LogLevelInfo,
			Source:    model.SourceJob,
			Event:     model.EventJobStarted,
			Message:   fmt.Sprintf("Job started: %s", jobType),
			JobID:     jobID,
			Timestamp: time.Now().UTC(),
			Metadata: map[string]any{
				"jobType":     jobType,
				"triggeredBy": triggeredBy,
				"summary":     summary,
			},
		}
		_ = ls.Write(entry) // Log errors are not critical
	}

	return &JobContext{
		JobID:       jobID,
		jobsService: js,
		logService:  ls,
		jobType:     jobType,
		triggeredBy: triggeredBy,
	}, nil
}

// Complete marks the job as successfully completed
func (jc *JobContext) Complete(summary string) error {
	// Log completion
	if jc.logService != nil {
		entry := model.LogEntry{
			Level:     model.LogLevelInfo,
			Source:    model.SourceJob,
			Event:     model.EventJobFinished,
			Message:   fmt.Sprintf("Job completed: %s", jc.jobType),
			JobID:     jc.JobID,
			Timestamp: time.Now().UTC(),
		}
		_ = jc.logService.Write(entry)
	}

	return jc.jobsService.Finish(jc.JobID, model.JobStatusCompleted, summary, "")
}

// Fail marks the job as failed
func (jc *JobContext) Fail(errMsg string) error {
	// Log failure
	if jc.logService != nil {
		entry := model.LogEntry{
			Level:     model.LogLevelError,
			Source:    model.SourceJob,
			Event:     model.EventJobFailed,
			Message:   fmt.Sprintf("Job failed: %s - %s", jc.jobType, errMsg),
			JobID:     jc.JobID,
			Timestamp: time.Now().UTC(),
		}
		_ = jc.logService.Write(entry)
	}

	return jc.jobsService.Finish(jc.JobID, model.JobStatusFailed, "", errMsg)
}

// Log emits a log entry linked to this job
func (jc *JobContext) Log(event, message, level string) error {
	if jc.logService == nil {
		return nil
	}

	entry := model.LogEntry{
		Level:     level,
		Source:    model.SourceJob,
		Event:     event,
		Message:   message,
		JobID:     jc.JobID,
		Timestamp: time.Now().UTC(),
	}

	return jc.logService.Write(entry)
}
