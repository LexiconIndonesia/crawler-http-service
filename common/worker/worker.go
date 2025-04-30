package worker

import (
	"context"
)

// JobID represents a unique identifier for a worker job
type JobID string

// Worker defines the interface for a worker that can process jobs
type Worker interface {
	// Start initializes the worker
	Start(ctx context.Context) error

	// Stop gracefully stops the worker
	Stop(ctx context.Context) error

	// CancelJob cancels a specific job by ID
	CancelJob(jobID JobID) error
}

// Job represents a task to be executed by a worker
type Job interface {
	// GetID returns the job's unique identifier
	GetID() JobID

	// Execute runs the job
	Execute(ctx context.Context) error

	// Cancel stops the job
	Cancel() error
}

// JobStatus represents the current state of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)
