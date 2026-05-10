// Package workpool provides a bounded goroutine pool for running tasks concurrently with a fixed number of workers.
package workpool

import (
	"fmt"
	"sync/atomic"
)

// Task is a zero-argument, zero-return function that can be added to the work pool.
// Panics inside a Task are recovered by the worker and printed to stdout; they are not propagated to the caller.
type Task func()

// WorkPool is a bounded goroutine pool for executing tasks concurrently with a fixed number of workers.
//
// Lifecycle: create with New, add tasks with Do (multiple times), then call Run to wait for completion.
// After Run returns, the pool is closed and further Do calls are silently ignored.
// Workers are started immediately upon pool creation. The zero value is not usable.
type WorkPool struct {
	queue   chan Task
	closed  bool
	workers int
	running int32
}

// New creates a new WorkPool with the specified number of workers.
//
// The workers parameter specifies the number of concurrent workers; a minimum of 1 is enforced.
// The default buffer size for the task queue is workers*100+1, which is large enough to keep workers busy
// without blocking the caller on typical workloads. optionalBufferSize overrides the default buffer size;
// use 0 or 1 for unbuffered-like behavior.
// Workers are started immediately and begin processing tasks from the queue.
func New(workers int, optionalBufferSize ...int) *WorkPool {
	if workers < 1 {
		workers = 1
	}
	bufferSize := workers*100 + 1
	if len(optionalBufferSize) > 0 {
		bufferSize = optionalBufferSize[0]
	}

	wp := &WorkPool{
		workers: workers,
		queue:   make(chan Task, bufferSize), // Buffered channel to prevent blocking
	}

	// Initialize the workers as soon as the pool is created
	wp.startWorkers()

	return wp
}

// Do adds a task to the work pool for execution by a worker.
//
// Do blocks if the queue channel is full, causing the caller to block until a worker drains a slot.
// The method returns the receiver to allow method chaining.
// After Run has been called and the pool is closed, Do is a no-op and returns immediately without adding the task.
func (wp *WorkPool) Do(task Task) *WorkPool {
	if wp.closed {
		return wp
	}

	atomic.AddInt32(&wp.running, 1) // Increment running tasks
	wp.queue <- task                // Add task to the queue
	return wp
}

// Run waits for all tasks to complete and shuts down the pool.
//
// Run busy-waits until the running task counter reaches zero (intentional; see nolint:revive comment),
// then closes the queue channel, which causes all workers to exit their range loops.
// After Run returns, the pool cannot be reused and further Do calls will be silently ignored.
func (wp *WorkPool) Run() {
	if wp.closed {
		return
	}

	//nolint:revive // wait until all tasks are processed
	for atomic.LoadInt32(&wp.running) > 0 {
	}
	wp.closed = true
	close(wp.queue)
}

// IsRunning reports whether there are any tasks queued or currently executing.
//
// IsRunning reflects the atomic counter of running tasks; it returns true if at least one task is queued or executing.
func (wp *WorkPool) IsRunning() bool {
	return atomic.LoadInt32(&wp.running) > 0
}

// startWorkers initializes the workers and starts processing tasks
func (wp *WorkPool) startWorkers() {
	for range wp.workers {
		go wp.worker()
	}
}

// worker processes tasks from the queue
func (wp *WorkPool) worker() {
	for task := range wp.queue {
		func(task Task) {
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("WARNING: WorkPool worker recovered from panic in task:", r)
				}
			}()

			task()
		}(task)
		atomic.AddInt32(&wp.running, -1) // Decrement running tasks when done
	}
}
