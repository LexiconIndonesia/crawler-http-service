package work

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// task implements the unified generic Executor interface
type task[T any] struct {
	ID           string
	execute      func(ctx context.Context) (T, error)
	errorHandler func(error)
	timeout      time.Duration
}

// TaskOption represents a functional option for task configuration
type TaskOption[T any] func(*task[T])

// WithID sets a custom ID for the task
func WithID[T any](id string) TaskOption[T] {
	return func(t *task[T]) {
		t.ID = id
	}
}

// WithErrorHandler sets a custom error handler for the task
func WithErrorHandler[T any](handler func(error)) TaskOption[T] {
	return func(t *task[T]) {
		t.errorHandler = handler
	}
}

// WithTimeout sets a custom timeout for the task
func WithTimeout[T any](timeout time.Duration) TaskOption[T] {
	return func(t *task[T]) {
		t.timeout = timeout
	}
}

// NewTask creates a new context-aware task with type safety and optional configuration
func NewTask[T any](
	execute func(ctx context.Context) (T, error),
	options ...TaskOption[T],
) (Executor[T], error) {
	// Generate UUID by default
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	// Create task with defaults
	t := &task[T]{
		ID:      id.String(),
		execute: execute,
		timeout: 0, // Use pool default
	}

	// Apply options
	for _, opt := range options {
		opt(t)
	}

	return t, nil
}

// MustNewTask creates a new task and panics on error (for cases where ID generation should never fail)
func MustNewTask[T any](
	execute func(ctx context.Context) (T, error),
	options ...TaskOption[T],
) Executor[T] {
	task, err := NewTask(execute, options...)
	if err != nil {
		panic(err)
	}
	return task
}

// SimpleTask creates a task that returns no meaningful result (just completion status)
func SimpleTask(
	execute func(ctx context.Context) error,
	options ...TaskOption[struct{}],
) (Executor[struct{}], error) {
	return NewTask[struct{}](
		func(ctx context.Context) (struct{}, error) {
			err := execute(ctx)
			return struct{}{}, err
		},
		options...,
	)
}

// MustSimpleTask creates a simple task and panics on error
func MustSimpleTask(
	execute func(ctx context.Context) error,
	options ...TaskOption[struct{}],
) Executor[struct{}] {
	task, err := SimpleTask(execute, options...)
	if err != nil {
		panic(err)
	}
	return task
}

// ExecutorID returns the task ID
func (t *task[T]) ExecutorID() string {
	return t.ID
}

// Execute runs the task with context support and returns the result directly
func (t *task[T]) Execute(ctx context.Context) (T, error) {
	return t.execute(ctx)
}

// OnError handles task errors
func (t *task[T]) OnError(err error) {
	if t.errorHandler != nil {
		t.errorHandler(err)
	}
}

// Timeout returns the task timeout duration (0 means use pool default)
func (t *task[T]) Timeout() time.Duration {
	return t.timeout
}
