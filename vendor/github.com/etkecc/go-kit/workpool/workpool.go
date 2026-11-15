// Package workpool provides a bounded goroutine pool for running tasks concurrently with a fixed number of workers.
package workpool

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Task is a no-arg, no-return func; a panic inside is recovered and printed, never returned to the caller.
type Task func()

// WorkPool runs tasks across a fixed set of workers; safe for concurrent Do and Run, zero value unusable.
type WorkPool struct {
	queue   chan Task
	wg      sync.WaitGroup
	mu      sync.RWMutex
	closed  bool
	workers int
	running int32
}

// New starts workers immediately (clamped to min 1); queue buffers workers*100+1 unless optionalBufferSize overrides.
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

// Do queues task and returns the receiver to chain; after Run closes the pool, Do silently drops the task.
func (wp *WorkPool) Do(task Task) *WorkPool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	if wp.closed {
		return wp
	}

	atomic.AddInt32(&wp.running, 1) // running: lock-free counter for IsRunning
	wp.wg.Add(1)                    // wg: what Run parks on
	wp.queue <- task
	return wp
}

// Run blocks until all tasks finish, then closes the queue; a mutex guard makes concurrent Run calls a safe no-op.
func (wp *WorkPool) Run() {
	wp.mu.Lock()
	if wp.closed {
		wp.mu.Unlock()
		return
	}
	wp.closed = true
	wp.mu.Unlock()

	wp.wg.Wait()
	close(wp.queue)
}

// IsRunning reports whether any task is still queued or in flight, straight off the atomic counter.
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
		atomic.AddInt32(&wp.running, -1)
		wp.wg.Done()
	}
}
