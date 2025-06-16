package work

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidWorkerCount = errors.New("invalid worker count")
	ErrInvalidChannelSize = errors.New("invalid channel size")
	ErrPoolStopped        = errors.New("worker pool has been stopped")
	ErrTaskTimeout        = errors.New("task execution timeout")
	ErrAddTaskTimeout     = errors.New("add task timeout")
)

// TaskResult represents the result of a task execution with type safety
type TaskResult[T any] struct {
	TaskID    string
	Result    T
	Error     error
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
}

// IsSuccess returns true if the task completed successfully
func (tr *TaskResult[T]) IsSuccess() bool {
	return tr.Error == nil
}

// Executor interface for all tasks - now generic for type safety
type Executor[T any] interface {
	ExecutorID() string
	Execute(ctx context.Context) (T, error) // Combined execution and result in one method
	OnError(error)
	Timeout() time.Duration // Optional timeout for the task (0 means use pool default)
}

// PoolConfig holds configuration for the worker pool
type PoolConfig struct {
	NumWorkers      int
	TaskChannelSize int
	ResultChanSize  int           // Buffer size for result channel
	TaskTimeout     time.Duration // Default timeout for tasks
	ShutdownTimeout time.Duration // Timeout for graceful shutdown
}

// DefaultPoolConfig returns a sensible default configuration
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		NumWorkers:      10,
		TaskChannelSize: 100,
		ResultChanSize:  100,
		TaskTimeout:     30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

// Pool is an enhanced worker pool implementation with generics
type Pool[T any] struct {
	config   PoolConfig
	tasks    chan Executor[T]
	results  chan TaskResult[T]
	quit     chan struct{}
	wg       sync.WaitGroup
	once     sync.Once
	stopOnce sync.Once

	// Metrics
	activeWorkers  int64
	tasksQueued    int64
	tasksCompleted int64

	// State
	started bool
	stopped bool
	mu      sync.RWMutex
}

// NewWorkerPool creates a new enhanced worker pool
func NewWorkerPool[T any](numWorkers int, taskChannelSize int) (*Pool[T], error) {
	config := PoolConfig{
		NumWorkers:      numWorkers,
		TaskChannelSize: taskChannelSize,
		ResultChanSize:  numWorkers * 2,
		TaskTimeout:     30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
	return NewWorkerPoolWithConfig[T](config)
}

// NewWorkerPoolWithConfig creates a new enhanced worker pool with custom configuration
func NewWorkerPoolWithConfig[T any](config PoolConfig) (*Pool[T], error) {
	if config.NumWorkers <= 0 {
		return nil, ErrInvalidWorkerCount
	}

	if config.TaskChannelSize < 0 {
		return nil, ErrInvalidChannelSize
	}

	if config.ResultChanSize < 0 {
		config.ResultChanSize = config.NumWorkers * 2 // Reasonable default
	}

	if config.TaskTimeout <= 0 {
		config.TaskTimeout = 30 * time.Second
	}

	if config.ShutdownTimeout <= 0 {
		config.ShutdownTimeout = 10 * time.Second
	}

	return &Pool[T]{
		config:  config,
		tasks:   make(chan Executor[T], config.TaskChannelSize),
		results: make(chan TaskResult[T], config.ResultChanSize),
		quit:    make(chan struct{}),
	}, nil
}

// Start starts the worker pool
func (p *Pool[T]) Start(ctx context.Context, workerPoolID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return // Already started
	}

	if p.stopped {
		log.Error().Msg("Cannot start a stopped pool")
		return
	}

	p.once.Do(func() {
		p.started = true
		p.startWorkers(ctx, workerPoolID)
		log.Info().
			Str("workerPoolID", workerPoolID).
			Int("numWorkers", p.config.NumWorkers).
			Msg("Worker pool started")
	})
}

// Stop gracefully stops the worker pool
func (p *Pool[T]) Stop() {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}
	p.stopped = true
	p.mu.Unlock()

	p.stopOnce.Do(func() {
		close(p.quit)

		// Close task channel to signal no more tasks
		close(p.tasks)

		// Wait for workers to finish with timeout
		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			log.Info().Msg("All workers stopped gracefully")
		case <-time.After(p.config.ShutdownTimeout):
			log.Warn().Dur("timeout", p.config.ShutdownTimeout).Msg("Shutdown timeout exceeded")
		}

		// Close results channel
		close(p.results)
	})
}

// AddTask adds a task to the pool with blocking behavior
func (p *Pool[T]) AddTask(ctx context.Context, task Executor[T]) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.stopped {
		return ErrPoolStopped
	}

	select {
	case p.tasks <- task:
		atomic.AddInt64(&p.tasksQueued, 1)
		return nil
	case <-p.quit:
		return ErrPoolStopped
	case <-ctx.Done():
		return ctx.Err()
	}
}

// AddTaskWithTimeout adds a task with a specific timeout
func (p *Pool[T]) AddTaskWithTimeout(task Executor[T], timeout time.Duration) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.stopped {
		return ErrPoolStopped
	}

	select {
	case p.tasks <- task:
		atomic.AddInt64(&p.tasksQueued, 1)
		return nil
	case <-p.quit:
		return ErrPoolStopped
	case <-time.After(timeout):
		return ErrAddTaskTimeout
	}
}

// AddTaskNonBlocking adds a task without blocking
func (p *Pool[T]) AddTaskNonBlocking(task Executor[T]) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.stopped {
		return ErrPoolStopped
	}

	select {
	case p.tasks <- task:
		atomic.AddInt64(&p.tasksQueued, 1)
		return nil
	case <-p.quit:
		return ErrPoolStopped
	default:
		return errors.New("task queue is full")
	}
}

// Results returns the results channel
func (p *Pool[T]) Results() <-chan TaskResult[T] {
	return p.results
}

// Stats returns pool statistics
func (p *Pool[T]) Stats() PoolStats {
	return PoolStats{
		ActiveWorkers:  atomic.LoadInt64(&p.activeWorkers),
		TasksQueued:    atomic.LoadInt64(&p.tasksQueued),
		TasksCompleted: atomic.LoadInt64(&p.tasksCompleted),
		TasksInQueue:   int64(len(p.tasks)),
	}
}

// PoolStats holds statistics about the pool
type PoolStats struct {
	ActiveWorkers  int64
	TasksQueued    int64
	TasksCompleted int64
	TasksInQueue   int64
}

// startWorkers starts the worker goroutines
func (p *Pool[T]) startWorkers(ctx context.Context, poolID string) {
	for i := 0; i < p.config.NumWorkers; i++ {
		p.wg.Add(1)
		go func(workerID int) {
			defer p.wg.Done()
			atomic.AddInt64(&p.activeWorkers, 1)
			defer atomic.AddInt64(&p.activeWorkers, -1)

			log.Info().
				Str("workerPoolID", poolID).
				Int("workerID", workerID).
				Msg("Worker started")

			for {
				select {
				case <-ctx.Done():
					log.Info().
						Str("workerPoolID", poolID).
						Int("workerID", workerID).
						Msg("Worker stopped due to context cancellation")
					return
				case <-p.quit:
					log.Info().
						Str("workerPoolID", poolID).
						Int("workerID", workerID).
						Msg("Worker stopped due to pool shutdown")
					return
				case task, ok := <-p.tasks:
					if !ok {
						log.Info().
							Str("workerPoolID", poolID).
							Int("workerID", workerID).
							Msg("Worker stopped - task channel closed")
						return
					}

					p.executeTask(ctx, task, workerID, poolID)
				}
			}
		}(i)
	}
}

// executeTask executes a single task with proper error handling and timeout
func (p *Pool[T]) executeTask(ctx context.Context, task Executor[T], workerID int, poolID string) {
	taskID := task.ExecutorID()
	startTime := time.Now()

	// Determine timeout for this task
	timeout := p.config.TaskTimeout
	if taskTimeout := task.Timeout(); taskTimeout > 0 {
		timeout = taskTimeout
	}

	// Create context with timeout
	taskCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.Debug().
		Str("workerPoolID", poolID).
		Int("workerID", workerID).
		Str("taskID", taskID).
		Dur("timeout", timeout).
		Msg("Executing task")

	// Execute task - now returns result and error in one call
	result, err := task.Execute(taskCtx)
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Handle timeout error specifically - check both context types
	if err != nil && (errors.Is(err, context.DeadlineExceeded) || taskCtx.Err() == context.DeadlineExceeded) {
		err = ErrTaskTimeout
	}

	// Create task result with timing information
	taskResult := TaskResult[T]{
		TaskID:    taskID,
		Result:    result,
		Error:     err,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  duration,
	}

	// Call error handler if there was an error
	if err != nil {
		task.OnError(err)
	}

	// Send result with timeout to prevent indefinite blocking but avoid dropping results immediately
	select {
	case p.results <- taskResult:
		// Result sent successfully
	case <-time.After(1 * time.Second):
		// If we can't send the result within a reasonable time, log and drop it
		log.Warn().
			Str("taskID", taskID).
			Msg("Result channel full after timeout, dropping result")
	case <-p.quit:
		// Pool is shutting down, don't block
		log.Debug().
			Str("taskID", taskID).
			Msg("Pool shutting down, dropping result")
	}

	atomic.AddInt64(&p.tasksCompleted, 1)

	log.Debug().
		Str("workerPoolID", poolID).
		Int("workerID", workerID).
		Str("taskID", taskID).
		Dur("duration", duration).
		Bool("success", err == nil).
		Msg("Task completed")
}
