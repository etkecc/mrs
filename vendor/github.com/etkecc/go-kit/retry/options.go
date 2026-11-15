package retry

import "time"

// DefaultMaxRetries is 3 total attempts (1 initial + 2 retries) when no WithMaxRetries option is given.
const DefaultMaxRetries = 3

// DefaultDelayStep is 1s: the linear backoff between attempt i and i+1 is delayStep*(i+1) before jitter.
const DefaultDelayStep = 1 * time.Second

// DefaultJitter is true: full AWS jitter picks the delay randomly in [0, base), base = delayStep*(i+1).
const DefaultJitter = true

// Option is a functional-options closure that mutates a *Retry; New applies them in order.
type Option func(*Retry)

// WithMaxRetries sets the maximum attempts (1 initial + retries); New clamps values below 1 to 1.
func WithMaxRetries(maxRetries int) Option {
	return func(r *Retry) {
		r.maxRetries = maxRetries
	}
}

// WithDelayStep sets the linear backoff base: delay between attempt i and i+1 is delayStep*(i+1) before jitter.
func WithDelayStep(delayStep time.Duration) Option {
	return func(r *Retry) {
		r.delayStep = delayStep
	}
}

// WithJitter toggles AWS-canonical full jitter (random in [0, base)) vs the deterministic delayStep*(i+1) delay.
func WithJitter(jitter bool) Option {
	return func(r *Retry) {
		r.jitter = jitter
	}
}

// WithRetryIf sets a predicate deciding if an error is retryable; false stops retries immediately, nil is ignored.
func WithRetryIf(predicate func(error) bool) Option {
	return func(r *Retry) {
		if predicate != nil {
			r.retryIf = predicate
		}
	}
}
