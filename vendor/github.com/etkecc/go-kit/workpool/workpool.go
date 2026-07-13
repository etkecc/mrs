// Package workpool provides a bounded goroutine pool for running tasks concurrently with a fixed number of workers.
package workpool

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Task is a no-arg, no-return func you hand to the pool. A panic inside one is recovered by the
// worker and printed to stdout; it does NOT come back to you, so a Task that can fail should
// report its outcome some other way.
type Task func()

// WorkPool runs tasks across a fixed set of workers, so a flood of work doesn't turn into a flood
// of goroutines.
//
// Lifecycle: New, then Do as many times as you need, then Run to block until the queue drains.
// Workers start the instant New returns. After Run the pool is closed and any further Do is
// silently dropped (see Do). The zero value is not usable.
//
// It's built single-producer: queue all the work, then Run. Do is fine to call concurrently with
// other Do calls, and Run is safe to call more than once (a second Run is a no-op). The one thing
// the single-producer rule forbids is Do racing Run: a Do that slips past the closed check exactly
// as Run starts can still Add to a draining WaitGroup, and that's on the caller.
type WorkPool struct {
	queue   chan Task
	wg      sync.WaitGroup
	closed  atomic.Bool
	workers int
	running int32
}

// New starts a pool of workers (minimum 1, it clamps up) chewing through tasks right away. The
// queue buffers workers*100+1 by default, deep enough that Do rarely blocks on normal loads; pass
// optionalBufferSize to override it (0 or 1 for near-unbuffered).
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

// Do queues task and returns the receiver so calls chain. It blocks while the queue is full, until
// a worker frees a slot. The trap: once Run has closed the pool, Do silently drops the task, no
// error, no panic, just gone. Queue everything before you call Run.
func (wp *WorkPool) Do(task Task) *WorkPool {
	if wp.closed.Load() {
		return wp
	}

	atomic.AddInt32(&wp.running, 1) // running: lock-free counter for IsRunning
	wp.wg.Add(1)                    // wg: what Run parks on
	wp.queue <- task
	return wp
}

// Run blocks until every queued task has finished, then closes the queue so the workers exit. It
// flips the pool closed the instant it starts (via CAS), so a second or concurrent Run is a safe
// no-op instead of a double-close panic, and a late Do is rejected rather than added to a draining
// WaitGroup. Then it parks on the WaitGroup rather than spinning, so it won't fight your own workers
// for a core while it waits. Single-use: once Run returns the pool is spent, build a new one.
func (wp *WorkPool) Run() {
	if !wp.closed.CompareAndSwap(false, true) {
		return
	}

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
