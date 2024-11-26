// package workpool provides a simple work pool implementation
package workpool

import "sync"

// Task is a function that can be added to the work pool
type Task func()

// WorkPool is a simple work pool implementation
type WorkPool struct {
	mu      sync.Mutex
	queue   []Task
	workers int
	wg      sync.WaitGroup
}

// New creates a new work pool with the specified number of workers
func New(workers int) *WorkPool {
	if workers < 1 {
		workers = 1
	}

	wp := &WorkPool{
		workers: workers,
		queue:   make([]Task, 0),
	}
	return wp
}

// Do adds a task to the work pool
//
//nolint:unparam // that's for users convenience, not for the library
func (wp *WorkPool) Do(task Task) *WorkPool {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.queue = append(wp.queue, task)
	return wp
}

// Run starts the work pool and waits for all tasks to complete
func (wp *WorkPool) Run() {
	wp.mu.Lock()
	if len(wp.queue) == 0 {
		wp.mu.Unlock()
		return
	}

	tasks := make(chan Task, len(wp.queue))
	wp.wg.Add(wp.workers)

	for i := 0; i < wp.workers; i++ {
		go wp.worker(tasks)
	}

	for _, task := range wp.queue {
		tasks <- task
	}
	close(tasks)
	wp.mu.Unlock()

	wp.wg.Wait()
}

func (wp *WorkPool) worker(tasks <-chan Task) {
	defer func() {
		_ = recover() //nolint:errcheck // ignore panic
		wp.wg.Done()
	}()

	for task := range tasks {
		task()
	}
}
