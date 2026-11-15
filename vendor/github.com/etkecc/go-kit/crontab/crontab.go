// Package crontab runs a five-field cron schedule at minute granularity; DST fall-back can double-fire a job.
package crontab

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// ErrClosed is returned by AddJob once Shutdown has been called.
var ErrClosed = errors.New("crontab: closed")

// Job is the function a schedule runs; it takes no args and returns nothing, closing over whatever state it needs.
type Job func()

// Option configures a Crontab at construction; see WithLocation, WithOverlap, and WithPanicHandler.
type Option func(*Crontab)

type job struct {
	sched   *schedule
	fn      Job
	spec    string      // original schedule text, handed to the panic handler
	running atomic.Bool // non-overlap guard: true while a fire is in flight
	last    time.Time   // last minute this job fired, as an absolute instant; see runDue
}

// Crontab is an in-process cron scheduler; New starts it, AddJob adds jobs, Shutdown stops it; safe for concurrent use.
type Crontab struct {
	loc     *time.Location
	overlap bool
	onPanic func(spec string, recovered any)
	now     func() time.Time // clock seam; time.Now in production, a fake in tests

	mu      sync.RWMutex
	jobs    []*job
	stop    chan struct{}
	once    sync.Once
	stopped atomic.Bool
	wg      sync.WaitGroup
}

// New builds a Crontab and starts its ticker; defaults UTC, skip-on-overlap, stderr panic log, all overridable.
func New(opts ...Option) *Crontab {
	c := &Crontab{
		loc:     time.UTC,
		onPanic: defaultPanicHandler,
		now:     time.Now,
		stop:    make(chan struct{}),
	}
	for _, opt := range opts {
		opt(c)
	}
	go c.run()
	return c
}

// WithLocation runs every schedule in loc instead of UTC; under DST a job can fire twice on fall-back night.
func WithLocation(loc *time.Location) Option {
	return func(c *Crontab) { c.loc = loc }
}

// WithOverlap lets a job run concurrently with itself instead of skipping the tick while a previous run is in flight.
func WithOverlap() Option {
	return func(c *Crontab) { c.overlap = true }
}

// WithPanicHandler replaces the default stderr handler with h; keep it quick, it runs on the job's own goroutine.
func WithPanicHandler(h func(spec string, recovered any)) Option {
	return func(c *Crontab) { c.onPanic = h }
}

// AddJob parses spec and registers fn; returns a parse error, or ErrClosed once the scheduler has shut down.
func (c *Crontab) AddJob(spec string, fn Job) error {
	sched, err := parse(spec)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped.Load() { // checked under the lock Shutdown flips it under, so no TOCTOU
		return ErrClosed
	}
	c.jobs = append(c.jobs, &job{sched: sched, fn: fn, spec: spec})
	return nil
}

// MustAddJob is AddJob that panics on error; use only for compile-time-constant schedules, never user-supplied specs.
func (c *Crontab) MustAddJob(spec string, fn Job) {
	if err := c.AddJob(spec, fn); err != nil {
		panic(err)
	}
}

// run aligns to the next minute then ticks every minute until stop closes, using a local ticker that outlives nothing.
func (c *Crontab) run() {
	start := c.now()
	first := start.Truncate(time.Minute).Add(time.Minute)
	timer := time.NewTimer(first.Sub(start))
	select {
	case <-c.stop:
		timer.Stop()
		return
	case <-timer.C:
	}
	c.runDue(c.now())

	tk := time.NewTicker(time.Minute)
	defer tk.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-tk.C:
			c.runDue(c.now())
		}
	}
}

// runDue fires jobs matching minute; single-goroutine only (j.last write unsynchronized), checks stopped under c.mu.
func (c *Crontab) runDue(now time.Time) {
	minute := now.In(c.loc).Truncate(time.Minute)
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.stopped.Load() {
		return
	}
	for _, j := range c.jobs {
		if !j.sched.match(minute) {
			continue
		}
		if j.last.Equal(minute) { // this exact instant already fired
			continue
		}
		if !c.overlap && !j.running.CompareAndSwap(false, true) {
			continue // previous run still in flight
		}
		j.last = minute
		c.wg.Add(1)
		go c.fire(j)
	}
}

// fire runs one job behind panic-safe guards: recover, clear the non-overlap flag, and wg.Done so Shutdown cannot hang.
func (c *Crontab) fire(j *job) {
	defer c.wg.Done()
	if !c.overlap {
		defer j.running.Store(false)
	}
	defer func() {
		if r := recover(); r != nil {
			// onPanic is user code; if it panics too, swallow that so a panicking handler cannot crash the process either.
			defer func() { _ = recover() }() //nolint:errcheck // deliberately swallow a handler panic
			c.onPanic(j.spec, r)
		}
	}()
	j.fn()
}

// Shutdown stops the scheduler, waits bounded by ctx for jobs to finish, returns ctx.Err() on timeout; idempotent.
func (c *Crontab) Shutdown(ctx context.Context) error {
	c.once.Do(func() {
		// Flip stopped under the lock runDue reads it under, so in-flight dispatch finishes wg.Add before the wait starts.
		c.mu.Lock()
		c.stopped.Store(true)
		c.mu.Unlock()
		close(c.stop)
	})
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// defaultPanicHandler writes a recovered job panic to stderr; the fallback when no WithPanicHandler is set.
func defaultPanicHandler(spec string, recovered any) {
	fmt.Fprintln(os.Stderr, "crontab: recovered panic in job", spec, recovered)
}
