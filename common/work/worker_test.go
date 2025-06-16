package work

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	tests := []struct {
		name            string
		numWorkers      int
		taskChannelSize int
		expectError     bool
	}{
		{"valid pool", 5, 10, false},
		{"zero workers", 0, 10, true},
		{"negative workers", -1, 10, true},
		{"negative channel size", 5, -1, true},
		{"zero channel size", 5, 0, false}, // This should be allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := NewWorkerPool[string](tt.numWorkers, tt.taskChannelSize)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if pool == nil {
				t.Error("Expected pool but got nil")
			}
		})
	}
}

func TestWorkerPoolBasicOperation(t *testing.T) {
	ctx := context.Background()
	pool, err := NewWorkerPool[string](2, 5)
	if err != nil {
		t.Fatal(err)
	}

	pool.Start(ctx, "test-pool")
	defer pool.Stop()

	// Create a simple task
	var executedCount int64
	task, err := NewTask[string](
		func(ctx context.Context) (string, error) {
			atomic.AddInt64(&executedCount, 1)
			return "test result", nil
		},
		WithErrorHandler[string](func(err error) {
			t.Errorf("Unexpected error: %v", err)
		}),
		WithTimeout[string](5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Add task
	if err := pool.AddTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	// Wait for result
	select {
	case result := <-pool.Results():
		if !result.IsSuccess() {
			t.Errorf("Task failed: %v", result.Error)
		}
		if result.Result != "test result" {
			t.Errorf("Expected 'test result', got '%s'", result.Result)
		}
		if atomic.LoadInt64(&executedCount) != 1 {
			t.Errorf("Expected 1 execution, got %d", executedCount)
		}
	case <-time.After(3 * time.Second):
		t.Error("Timeout waiting for result")
	}
}

func TestWorkerPoolConcurrency(t *testing.T) {
	ctx := context.Background()
	pool, err := NewWorkerPool[int](3, 10)
	if err != nil {
		t.Fatal(err)
	}

	pool.Start(ctx, "concurrency-test-pool")
	defer pool.Stop()

	const numTasks = 10
	var completedTasks int64

	// Add multiple tasks
	for i := 0; i < numTasks; i++ {
		taskNum := i
		task, err := NewTask[int](
			func(ctx context.Context) (int, error) {
				time.Sleep(50 * time.Millisecond) // Simulate work
				atomic.AddInt64(&completedTasks, 1)
				return taskNum * 2, nil
			},
			WithErrorHandler[int](func(err error) {
				t.Errorf("Task %d failed: %v", taskNum, err)
			}),
			WithTimeout[int](5*time.Second),
		)
		if err != nil {
			t.Fatal(err)
		}

		if err := pool.AddTask(ctx, task); err != nil {
			t.Fatal(err)
		}
	}

	// Collect results
	results := make([]int, 0, numTasks)
	for i := 0; i < numTasks; i++ {
		select {
		case result := <-pool.Results():
			if !result.IsSuccess() {
				t.Errorf("Task failed: %v", result.Error)
			} else {
				results = append(results, result.Result)
			}
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for results")
			break
		}
	}

	if len(results) != numTasks {
		t.Errorf("Expected %d results, got %d", numTasks, len(results))
	}

	if atomic.LoadInt64(&completedTasks) != numTasks {
		t.Errorf("Expected %d completed tasks, got %d", numTasks, completedTasks)
	}
}

func TestWorkerPoolTimeout(t *testing.T) {
	ctx := context.Background()
	pool, err := NewWorkerPool[string](1, 1)
	if err != nil {
		t.Fatal(err)
	}

	pool.Start(ctx, "timeout-test-pool")
	defer pool.Stop()

	// Create a task that will timeout
	task, err := NewTask[string](
		func(ctx context.Context) (string, error) {
			select {
			case <-time.After(2 * time.Second):
				return "should not complete", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		},
		WithErrorHandler[string](func(err error) {
			// Expected to be called with timeout error
		}),
		WithTimeout[string](100*time.Millisecond), // Short timeout
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := pool.AddTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	// Wait for result
	select {
	case result := <-pool.Results():
		if result.IsSuccess() {
			t.Error("Expected task to timeout")
		}
		// Should receive timeout error
		if !errors.Is(result.Error, ErrTaskTimeout) {
			t.Errorf("Expected timeout error, got: %v", result.Error)
		}
	case <-time.After(3 * time.Second):
		t.Error("Timeout waiting for result")
	}
}

func TestWorkerPoolGracefulShutdown(t *testing.T) {
	ctx := context.Background()
	pool, err := NewWorkerPool[string](2, 5)
	if err != nil {
		t.Fatal(err)
	}

	pool.Start(ctx, "shutdown-test-pool")

	// Add some tasks
	for i := 0; i < 3; i++ {
		task, err := NewTask[string](
			func(ctx context.Context) (string, error) {
				time.Sleep(100 * time.Millisecond)
				return fmt.Sprintf("task-%d", i), nil
			},
			WithErrorHandler[string](func(err error) {}),
			WithTimeout[string](5*time.Second),
		)
		if err != nil {
			t.Fatal(err)
		}

		if err := pool.AddTask(ctx, task); err != nil {
			t.Fatal(err)
		}
	}

	// Stop the pool
	pool.Stop()

	// Try to add a task after stopping
	task, err := NewTask[string](
		func(ctx context.Context) (string, error) {
			return "should not execute", nil
		},
		WithErrorHandler[string](func(err error) {}),
		WithTimeout[string](5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	err = pool.AddTask(ctx, task)
	if !errors.Is(err, ErrPoolStopped) {
		t.Errorf("Expected ErrPoolStopped, got: %v", err)
	}
}

func TestWorkerPoolStats(t *testing.T) {
	ctx := context.Background()
	pool, err := NewWorkerPool[string](2, 5)
	if err != nil {
		t.Fatal(err)
	}

	pool.Start(ctx, "stats-test-pool")
	defer pool.Stop()

	// Wait for workers to start
	time.Sleep(50 * time.Millisecond)

	// Initial stats
	stats := pool.Stats()
	if stats.ActiveWorkers != 2 {
		t.Errorf("Expected 2 active workers, got %d", stats.ActiveWorkers)
	}
	if stats.TasksQueued != 0 {
		t.Errorf("Expected 0 queued tasks, got %d", stats.TasksQueued)
	}

	// Add a task
	task, err := NewTask[string](
		func(ctx context.Context) (string, error) {
			time.Sleep(200 * time.Millisecond)
			return "test", nil
		},
		WithErrorHandler[string](func(err error) {}),
		WithTimeout[string](5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := pool.AddTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	// Check stats after adding task
	stats = pool.Stats()
	if stats.TasksQueued != 1 {
		t.Errorf("Expected 1 queued task, got %d", stats.TasksQueued)
	}

	// Wait for completion
	<-pool.Results()

	// Check final stats
	time.Sleep(100 * time.Millisecond) // Give time for stats to update
	stats = pool.Stats()
	if stats.TasksCompleted != 1 {
		t.Errorf("Expected 1 completed task, got %d", stats.TasksCompleted)
	}
}

func TestSimpleTask(t *testing.T) {
	ctx := context.Background()
	pool, err := NewWorkerPool[struct{}](1, 1)
	if err != nil {
		t.Fatal(err)
	}

	pool.Start(ctx, "simple-test-pool")
	defer pool.Stop()

	var executed bool
	task, err := SimpleTask(
		func(ctx context.Context) error {
			executed = true
			return nil
		},
		WithErrorHandler[struct{}](func(err error) {
			t.Errorf("Unexpected error: %v", err)
		}),
		WithTimeout[struct{}](5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := pool.AddTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	// Wait for result
	select {
	case result := <-pool.Results():
		if !result.IsSuccess() {
			t.Errorf("Simple task failed: %v", result.Error)
		}
		if !executed {
			t.Error("Task was not executed")
		}
	case <-time.After(3 * time.Second):
		t.Error("Timeout waiting for result")
	}
}

func TestAddTaskNonBlocking(t *testing.T) {
	ctx := context.Background()
	pool, err := NewWorkerPool[string](1, 1) // Small queue to test blocking
	if err != nil {
		t.Fatal(err)
	}

	pool.Start(ctx, "nonblocking-test-pool")
	defer pool.Stop()

	// Wait for worker to start
	time.Sleep(50 * time.Millisecond)

	// Fill the queue
	task1, err := NewTask[string](
		func(ctx context.Context) (string, error) {
			time.Sleep(500 * time.Millisecond) // Block worker
			return "task1", nil
		},
		WithErrorHandler[string](func(err error) {}),
		WithTimeout[string](5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := pool.AddTask(ctx, task1); err != nil {
		t.Fatal(err)
	}

	// Give time for task to start executing
	time.Sleep(50 * time.Millisecond)

	// Try to add another task (should fill queue)
	task2, err := NewTask[string](
		func(ctx context.Context) (string, error) {
			return "task2", nil
		},
		WithErrorHandler[string](func(err error) {}),
		WithTimeout[string](5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := pool.AddTaskNonBlocking(task2); err != nil {
		t.Fatal(err)
	}

	// Try to add a third task (should fail with queue full)
	task3, err := NewTask[string](
		func(ctx context.Context) (string, error) {
			return "task3", nil
		},
		WithErrorHandler[string](func(err error) {}),
		WithTimeout[string](5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}

	err = pool.AddTaskNonBlocking(task3)
	if err == nil {
		t.Error("Expected error when queue is full")
	}
	if err.Error() != "task queue is full" {
		t.Errorf("Expected 'task queue is full' error, got: %v", err)
	}
}
