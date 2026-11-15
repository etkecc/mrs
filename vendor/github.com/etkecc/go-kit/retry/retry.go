// Package retry provides a simple retry mechanism for Go with linear backoff and jitter.
package retry

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

var (
	// ErrNilRetry is returned by Do/DoCtx when called on a nil *Retry receiver.
	ErrNilRetry = errors.New("retry: nil retrier")
	// ErrNilContext is returned by DoCtx when a nil context is passed.
	ErrNilContext = errors.New("retry: nil context")
	// ErrNilFn is returned by Do/DoCtx when a nil function is passed.
	ErrNilFn = errors.New("retry: nil fn")
)

// DelayHinter lets an error carry a retry delay (e.g. Retry-After); use a VALUE receiver or errors.As misses it.
type DelayHinter interface {
	SuggestedRetryDelay() time.Duration
}

// hintError is a bare delay carrier returned by After; a value receiver keeps errors.As matching it either way.
type hintError struct {
	delay time.Duration
}

// Retry holds config for linear backoff with optional jitter; unsafe for concurrent use if options mutate during Do.
type Retry struct {
	maxRetries int
	delayStep  time.Duration
	retryIf    func(error) bool
	jitter     bool
}

// New creates a Retry with the Default* settings, applies options in order, and clamps maxRetries below 1 to 1.
func New(options ...Option) *Retry {
	r := &Retry{
		maxRetries: DefaultMaxRetries,
		delayStep:  DefaultDelayStep,
		jitter:     DefaultJitter,
		retryIf:    func(err error) bool { return err != nil },
	}

	for _, opt := range options {
		if opt != nil {
			opt(r)
		}
	}

	if r.maxRetries < 1 {
		r.maxRetries = 1
	}

	return r
}

// DoCtx runs fn with linear backoff and jitter until success, cancellation, retryIf rejection, or exhausted attempts.
func (r *Retry) DoCtx(ctx context.Context, fn func() error) error {
	if err := r.validate(ctx, fn); err != nil {
		return err
	}

	maxRetries := max(r.maxRetries, 1)

	var err error
	for i := range maxRetries {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err = fn(); err == nil {
			return nil
		}
		if !r.shouldRetry(err) {
			return err
		}
		if i == maxRetries-1 {
			break
		}
		if err := r.sleep(ctx, r.nextDelay(i, err)); err != nil {
			return err
		}
	}
	return err
}

// validate checks for nil receiver, ctx, and fn before the retry loop.
func (r *Retry) validate(ctx context.Context, fn func() error) error {
	if r == nil {
		return ErrNilRetry
	}
	if ctx == nil {
		return ErrNilContext
	}
	if fn == nil {
		return ErrNilFn
	}
	return nil
}

// shouldRetry reports whether err warrants another attempt.
func (r *Retry) shouldRetry(err error) bool {
	if r.retryIf == nil {
		return true
	}
	return r.retryIf(err)
}

// Do is DoCtx(context.Background(), fn); see DoCtx for the retry/backoff semantics.
func (r *Retry) Do(fn func() error) error {
	return r.DoCtx(context.Background(), fn)
}

// sleep waits for delay or until ctx is done, whichever comes first.
func (r *Retry) sleep(ctx context.Context, delay time.Duration) error {
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// calculateJitter returns a random delay in [0, base), AWS full jitter.
func (r *Retry) calculateJitter(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(delay.Nanoseconds())) //nolint:gosec // jitter does not require cryptographic randomness
}

// nextDelay computes the wait before the next retry; a DelayHinter overrides the linear step as a jitter-floor.
func (r *Retry) nextDelay(i int, err error) time.Duration {
	var dh DelayHinter
	if errors.As(err, &dh) {
		if hint := dh.SuggestedRetryDelay(); hint > 0 {
			if r.jitter {
				return hint + r.calculateJitter(hint/10)
			}
			return hint
		}
	}
	delay := r.delayStep * time.Duration(i+1)
	if r.jitter {
		delay = r.calculateJitter(delay)
	}
	return delay
}

// After returns an error honoring a server's Retry-After: DoCtx waits d instead of the linear step when retried.
func After(d time.Duration) error {
	return hintError{delay: d}
}

func (e hintError) Error() string {
	return "retry: suggested delay " + e.delay.String()
}

func (e hintError) SuggestedRetryDelay() time.Duration {
	return e.delay
}
