package httpclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"syscall"
)

var (
	// retryableStatuses are the HTTP statuses worth another attempt.
	retryableStatuses = map[int]struct{}{
		http.StatusTooManyRequests:     {},
		http.StatusInternalServerError: {},
		http.StatusBadGateway:          {},
		http.StatusServiceUnavailable:  {},
		http.StatusGatewayTimeout:      {},
	}
	// idempotentMethods are safe to replay per RFC 9110.
	idempotentMethods = map[string]struct{}{
		http.MethodGet:     {},
		http.MethodHead:    {},
		http.MethodOptions: {},
		http.MethodPut:     {},
		http.MethodDelete:  {},
		http.MethodTrace:   {},
	}
)

func isCallerCtxErr(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// defaultRetryIf: retry timeouts, retryable statuses, reset conns; else terminal. Method-blind; gated in transport.
func defaultRetryIf(err error) bool {
	if err == nil {
		return false
	}
	if isCallerCtxErr(err) {
		return false
	}
	if errors.Is(err, errAttemptTimeout) {
		return true
	}
	var rse retryableStatusError
	if errors.As(err, &rse) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET)
}
