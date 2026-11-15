package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"time"

	"github.com/etkecc/go-kit/retry"
)

// retryTransport is the retrying RoundTripper: turns a retryable status into an error the retrier classifies.
type retryTransport struct {
	base          http.RoundTripper
	retrier       *retry.Retry
	perAttempt    time.Duration
	nonIdem       bool
	maxRetryAfter time.Duration
	budget        RetryBudget
	onAttempt     func(AttemptInfo)
}

// retryableStatusError carries a retryable status out of an attempt, plus a parsed Retry-After delay (0 if absent).
type retryableStatusError struct {
	status int
	delay  time.Duration
}

// attemptLoop holds mutable state across one RoundTrip's attempts: held response, pending cancel, counter, terminal.
type attemptLoop struct {
	rt        *retryTransport
	req       *http.Request
	retryable bool

	resp       *http.Response
	prev       *http.Response
	prevCancel context.CancelFunc
	attempt    int
}

// cancelOnClose ties an attempt's cancel to body Close, closing body first so a read one returns its conn to the pool.
type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

// Value receivers, so both a value and pointer satisfy DelayHinter and errors.As matches however the error is wrapped.
func (e retryableStatusError) Error() string {
	return "httpclient: retryable status " + strconv.Itoa(e.status)
}

func (e retryableStatusError) SuggestedRetryDelay() time.Duration { return e.delay }

func (c *cancelOnClose) Close() error {
	err := c.ReadCloser.Close()
	c.cancel()
	return err
}

// RoundTrip runs the request through the retrier: method-and-body gate checked once, then each attempt via attemptLoop.
func (rt *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	_, idempotent := idempotentMethods[req.Method]
	retryable := rt.nonIdem || idempotent
	// A retryable request whose body can't rewind can't be replayed; refuse loud rather than resend a consumed reader.
	if retryable && req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		return nil, ErrNonReplayableBody
	}

	loop := &attemptLoop{rt: rt, req: req, retryable: retryable}
	if !retryable {
		// A non-idempotent request gets exactly one attempt; the classifier is method-blind, so the method gate lives here.
		return loop.result(loop.once())
	}
	return loop.result(rt.retrier.DoCtx(req.Context(), loop.once))
}

// once runs a single attempt: drain the prior response, gate on budget, then send.
func (l *attemptLoop) once() error {
	l.attempt++
	l.drainPrev()
	if l.attempt > 1 {
		if !l.rt.budget.Allow() {
			l.rt.budget.Record(false)
			return errBudgetExhausted
		}
		l.rt.budget.Record(true)
	}
	return l.send()
}

// drainPrev drains and closes the previous attempt's response, freeing its conn; the current response stays untouched.
func (l *attemptLoop) drainPrev() {
	if l.prev == nil {
		return
	}
	drainClose(l.prev, l.prevCancel)
	l.prev, l.prevCancel = nil, nil
}

// send issues one attempt under a fresh per-attempt context, rewinding the body on retries.
func (l *attemptLoop) send() error {
	attemptCtx, cancel := l.rt.attemptContext(l.req.Context())
	var reused bool
	traced := httptrace.WithClientTrace(attemptCtx, &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) { reused = info.Reused },
	})
	areq := l.req.WithContext(traced)
	if l.attempt > 1 && l.req.GetBody != nil {
		body, err := l.req.GetBody()
		if err != nil {
			cancel()
			return err
		}
		areq.Body = body
	}

	start := time.Now()
	r, rerr := l.rt.base.RoundTrip(areq) //nolint:bodyclose // returned to the caller (wrapped) or drained on retry
	elapsed := time.Since(start)

	if rerr != nil {
		cancel()
		normErr := l.rt.normalizeErr(l.req.Context(), attemptCtx, rerr)
		retrying := l.retryable && !isCallerCtxErr(normErr)
		l.rt.emit(l.req, l.attempt, 0, normErr, elapsed, retrying, reused)
		return normErr
	}
	return l.classify(r, cancel, elapsed, reused)
}

// classify decides a response: a retryable status within the Retry-After cap becomes an error for the retrier.
func (l *attemptLoop) classify(r *http.Response, cancel context.CancelFunc, elapsed time.Duration, reused bool) error {
	if _, ok := retryableStatuses[r.StatusCode]; l.retryable && ok {
		if delay, withinCap := l.rt.retryAfterDelay(r); withinCap {
			l.prev, l.prevCancel = r, cancel
			l.rt.emit(l.req, l.attempt, r.StatusCode, nil, elapsed, true, reused)
			return retryableStatusError{status: r.StatusCode, delay: delay}
		}
	}
	// Terminal response. l.prev is nil here: once drains it before every send, so nothing is held to clean up.
	wrapBodyCancel(r, cancel)
	l.resp = r
	l.rt.emit(l.req, l.attempt, r.StatusCode, nil, elapsed, false, reused)
	return nil
}

// result maps the retrier's outcome to RoundTrip's return: terminal wins, exhausted retryable status returns held live.
func (l *attemptLoop) result(err error) (*http.Response, error) {
	if l.resp != nil {
		return l.resp, nil
	}
	var rse retryableStatusError
	if errors.As(err, &rse) && l.prev != nil {
		wrapBodyCancel(l.prev, l.prevCancel)
		return l.prev, nil
	}
	if l.prev != nil {
		drainClose(l.prev, l.prevCancel)
	}
	return nil, err
}

// attemptContext derives the per-attempt context: perAttempt, or the caller's remaining deadline when that is sooner.
func (rt *retryTransport) attemptContext(parent context.Context) (context.Context, context.CancelFunc) {
	if rt.perAttempt <= 0 {
		return context.WithCancel(parent)
	}
	d := rt.perAttempt
	if dl, ok := parent.Deadline(); ok {
		if remaining := time.Until(dl); remaining < d {
			d = remaining
		}
	}
	return context.WithTimeout(parent, d)
}

// normalizeErr disambiguates a failed attempt: caller ctx death is terminal, per-attempt timeout retryable, else raw.
func (rt *retryTransport) normalizeErr(parent, attemptCtx context.Context, rerr error) error {
	if parent.Err() != nil {
		return parent.Err()
	}
	if attemptCtx.Err() != nil {
		return errAttemptTimeout
	}
	return rerr
}

// retryAfterDelay parses a Retry-After from a 429/503 and reports whether it's within maxRetryAfter.
func (rt *retryTransport) retryAfterDelay(r *http.Response) (delay time.Duration, withinCap bool) {
	delay = rt.parseRetryAfter(r)
	if delay > rt.maxRetryAfter {
		return 0, false
	}
	return delay, true
}

// parseRetryAfter reads a 429/503 Retry-After (delta-seconds or HTTP-date); an overflowing delta clamps past the cap.
func (rt *retryTransport) parseRetryAfter(r *http.Response) time.Duration {
	if r.StatusCode != http.StatusTooManyRequests && r.StatusCode != http.StatusServiceUnavailable {
		return 0
	}
	v := r.Header.Get("Retry-After")
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		if secs > int(rt.maxRetryAfter/time.Second)+1 {
			return rt.maxRetryAfter + time.Second
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

func (rt *retryTransport) emit(req *http.Request, attempt, status int, err error, elapsed time.Duration, retrying, reused bool) {
	if rt.onAttempt == nil {
		return
	}
	rt.onAttempt(AttemptInfo{
		Method:   req.Method,
		Host:     req.URL.Host,
		Attempt:  attempt,
		Status:   status,
		Err:      err,
		Elapsed:  elapsed,
		Retrying: retrying,
		Reused:   reused,
	})
}

// wrapBodyCancel replaces the response body with one that cancels the attempt context on Close.
func wrapBodyCancel(r *http.Response, cancel context.CancelFunc) {
	r.Body = &cancelOnClose{ReadCloser: r.Body, cancel: cancel}
}

// drainClose drains and closes a response body best-effort, then cancels its attempt context.
func drainClose(r *http.Response, cancel context.CancelFunc) {
	_, _ = io.Copy(io.Discard, r.Body) //nolint:errcheck // best-effort drain; a failure just costs the connection
	_ = r.Body.Close()
	cancel()
}
