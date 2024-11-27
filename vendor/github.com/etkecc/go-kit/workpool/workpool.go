// package workpool provides a simple work pool implementation
package workpool

import (
	"fmt"
	"sync/atomic"
)

// Task is a function that can be added to the work pool
type Task func()

// WorkPool is a simple work pool implementation
type WorkPool struct {
	queue   chan Task
	closed  bool
	workers int
	running int32
}

// New creates a new work pool with the specified number of workers
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

// Do adds a task to the work pool
func (wp *WorkPool) Do(task Task) *WorkPool {
	if wp.closed {
		return wp
	}

	atomic.AddInt32(&wp.running, 1) // Increment running tasks
	wp.queue <- task                // Add task to the queue
	return wp
}

// Run waits for all tasks to complete and shuts down the pool
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

// IsRunning returns true if there are any tasks in progress or in the queue
func (wp *WorkPool) IsRunning() bool {
	return atomic.LoadInt32(&wp.running) > 0
}

// startWorkers initializes the workers and starts processing tasks
func (wp *WorkPool) startWorkers() {
	for i := 0; i < wp.workers; i++ {
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
