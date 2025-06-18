package work

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LexiconIndonesia/crawler-http-service/common/db"
	"github.com/LexiconIndonesia/crawler-http-service/repository"
	"github.com/jackc/pgx/v5"
	redisv9 "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	workStateKeyPrefix = "work:state:"
	runningState       = "running"
	// workTimeout sets how long a work is considered running before it's considered stale.
	// This prevents works that died without proper cleanup from being stuck in 'running' state forever.
	workTimeout = 24 * time.Hour
)

// WorkManager manages the state of works in Redis.
type WorkManager struct {
	db *db.DB
}

// NewWorkManager creates a new WorkManager. The db parameter can be nil; in that case
// job state will only be stored in Redis.
func NewWorkManager(dbConn *db.DB) *WorkManager {
	return &WorkManager{
		db: dbConn,
	}
}

// getWorkKey returns the Redis key for a given work ID.
func (wm *WorkManager) getWorkKey(workID string) string {
	return fmt.Sprintf("%s%s", workStateKeyPrefix, workID)
}

// Start marks a work as running. It sets a key in Redis with an expiration.
// If the work is already running, it returns an error.
func (wm *WorkManager) Start(ctx context.Context, workID string) error {
	key := wm.getWorkKey(workID)
	// SetNX to prevent starting a work that is already running.
	ok, err := wm.db.Redis.SetNX(ctx, key, runningState, workTimeout)
	if err != nil {
		return fmt.Errorf("failed to start work %s: %w", workID, err)
	}
	if !ok {
		return fmt.Errorf("work %s is already running", workID)
	}

	if err := wm.updateJobStatus(ctx, workID, "started"); err != nil {
		log.Warn().Err(err).Str("workID", workID).Msg("failed to persist job status to DB")
	}

	return nil
}

// IsRunning checks if a work is currently marked as running.
func (wm *WorkManager) IsRunning(ctx context.Context, workID string) (bool, error) {
	key := wm.getWorkKey(workID)
	state, err := wm.db.Redis.Get(ctx, key)
	if err != nil {
		if errors.Is(err, redisv9.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get work state for %s: %w", workID, err)
	}
	return state == runningState, nil
}

// removeWork removes a work's state from Redis.
func (wm *WorkManager) removeWork(ctx context.Context, workID string) error {
	key := wm.getWorkKey(workID)
	err := wm.db.Redis.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to remove work %s: %w", workID, err)
	}
	return nil
}

// Complete marks a work as completed by removing its state from Redis.
func (wm *WorkManager) Complete(ctx context.Context, workID string) error {
	if err := wm.removeWork(ctx, workID); err != nil {
		return err
	}

	if err := wm.updateJobStatus(ctx, workID, "finished"); err != nil {
		log.Warn().Err(err).Str("workID", workID).Msg("failed to persist job completion to DB")
	}

	return nil
}

// Cancel marks a work as cancelled by removing its state from Redis.
func (wm *WorkManager) Cancel(ctx context.Context, workID string) error {
	if err := wm.removeWork(ctx, workID); err != nil {
		return err
	}

	if err := wm.updateJobStatus(ctx, workID, "cancelled"); err != nil {
		log.Warn().Err(err).Str("workID", workID).Msg("failed to persist job cancellation to DB")
	}

	return nil
}

// ListRunningWorks returns a slice of work IDs for all works currently marked as running.
// This can be used on startup to find and resume stale works.
// It uses SCAN to avoid blocking the Redis server.
func (wm *WorkManager) ListRunningWorks(ctx context.Context) ([]string, error) {
	var workIDs []string
	pattern := fmt.Sprintf("%s*", workStateKeyPrefix)

	iter := wm.db.Redis.GetClient().Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		workID := strings.TrimPrefix(key, workStateKeyPrefix)
		workIDs = append(workIDs, workID)
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan for running works in Redis: %w", err)
	}

	return workIDs, nil
}

// Resume checks if a work is running and extends its expiration if it is.
func (wm *WorkManager) Resume(ctx context.Context, workID string) (bool, error) {
	key := wm.getWorkKey(workID)
	state, err := wm.db.Redis.Get(ctx, key)
	if err != nil {
		if errors.Is(err, redisv9.Nil) {
			return false, nil // Not running
		}
		return false, fmt.Errorf("failed to get work state for %s: %w", workID, err)
	}

	if state == runningState {
		// If it's running, extend the expiration time.
		if err := wm.db.Redis.Set(ctx, key, runningState, workTimeout); err != nil {
			return true, fmt.Errorf("failed to extend work session for %s: %w", workID, err)
		}

		if err := wm.updateJobStatus(ctx, workID, "on_progress"); err != nil {
			log.Warn().Err(err).Str("workID", workID).Msg("failed to persist job resume to DB")
		}
		return true, nil
	}

	return false, nil
}

// updateJobStatus upserts the job record in the database.
// It is a no-op when database initialisation failed.
func (wm *WorkManager) updateJobStatus(ctx context.Context, workID, status string) error {
	if wm.db == nil || wm.db.Queries == nil {
		// DB not provided; skip persistence.
		return nil
	}

	// First try to update; if it affects no rows, create.
	if _, err := wm.db.Queries.UpdateJobStatus(ctx, repository.UpdateJobStatusParams{
		ID:     workID,
		Status: status,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Job row doesn't exist yet; create it.
			if _, createErr := wm.db.Queries.CreateJob(ctx, repository.CreateJobParams{
				ID:     workID,
				Status: status,
			}); createErr != nil {
				return fmt.Errorf("create job row: %w", createErr)
			}
			return nil
		}
		return fmt.Errorf("update job status: %w", err)
	}

	return nil
}
