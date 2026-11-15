package kit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// ErrUnknown is the fallback sentinel for an error you can't classify any better.
var ErrUnknown = errors.New("unknown error")

// maxMatrixErrorBody caps the response body MatrixErrorFrom reads, so a hostile server can't force an unbounded read.
const maxMatrixErrorBody = 64 << 10 // 64 KiB, roomy for any real Matrix error JSON

// AggregateError collects errors from a batch and reports them together; safe for concurrent use, zero value ready.
type AggregateError struct {
	mu     sync.RWMutex
	Errors []error
}

// Error joins every collected message with "; ". Empty aggregate, empty string. Implements error.
func (a *AggregateError) Error() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.Errors) == 0 {
		return ""
	}

	msgs := make([]string, 0, len(a.Errors))
	for _, err := range a.Errors {
		msgs = append(msgs, err.Error())
	}

	return strings.Join(msgs, "; ")
}

// Unwrap returns the underlying error slice for errors.Is/As; read-only, don't hold it across a concurrent Add.
func (a *AggregateError) Unwrap() []error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.Errors) == 0 {
		return nil
	}

	return a.Errors
}

// Is reports whether target matches any error in the bag via errors.Is, reaching through wrapped sentinels too.
func (a *AggregateError) Is(target error) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, err := range a.Errors {
		if errors.Is(err, target) {
			return true
		}
	}

	return false
}

// As finds the first collected error assignable to target via errors.As, reaching through wrapping too.
func (a *AggregateError) As(target any) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, err := range a.Errors {
		if errors.As(err, target) {
			return true
		}
	}

	return false
}

// Join adds the non-nil errors to the bag and returns the receiver, or nil if the bag is still empty.
func (a *AggregateError) Join(errs ...error) *AggregateError {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.Errors == nil {
		a.Errors = []error{}
	}

	for _, err := range errs {
		if err != nil {
			a.Errors = append(a.Errors, err)
		}
	}

	if len(a.Errors) == 0 {
		return nil
	}

	return a
}

// NewAggregateError builds an aggregate from the given errors, skipping nils, and returns nil if none were real.
func NewAggregateError(errs ...error) *AggregateError {
	agg := &AggregateError{}
	return agg.Join(errs...)
}

// ErrorResponse hands an error back to an HTTP client: status code plus a JSON body; the wrapped error stays hidden.
type ErrorResponse struct { //nolint:errname // ErrorResponse is a valid name
	StatusCode int    `json:"-"`     // HTTP status code, optional, not serialized
	Err        string `json:"error"` // Error message
	err        error  `json:"-"`     // underlying error, optional, not serialized
}

// Error returns the message. Implements error.
func (e ErrorResponse) Error() string {
	return e.Err
}

// Unwrap hands back the wrapped error so errors.Is/As can walk the chain.
func (e ErrorResponse) Unwrap() error {
	return e.err
}

// NewErrorResponse wraps err as a 400 by default (pass a status to override); a nil err becomes "unknown error".
func NewErrorResponse(err error, optionalStatusCode ...int) *ErrorResponse {
	statusCode := http.StatusBadRequest
	if len(optionalStatusCode) > 0 && optionalStatusCode[0] > 0 {
		statusCode = optionalStatusCode[0]
	}

	if err == nil {
		return &ErrorResponse{Err: ErrUnknown.Error(), StatusCode: statusCode}
	}

	return &ErrorResponse{Err: err.Error(), StatusCode: statusCode, err: err}
}

// MatrixError is the shape Matrix's Client-Server API speaks errors in: an errcode plus a human message.
type MatrixError struct {
	Code string `json:"errcode"`
	Err  string `json:"error"`
}

// Error returns the human-readable message. Implements error.
func (e MatrixError) Error() string {
	return e.Err
}

// NewMatrixError builds one from a standard Matrix errcode and a message.
func NewMatrixError(code, err string) *MatrixError {
	return &MatrixError{Code: code, Err: err}
}

// MatrixErrorFrom decodes a MatrixError from a JSON body, nil on a nil reader; bad JSON becomes M_UNKNOWN, read capped.
func MatrixErrorFrom(r io.Reader) *MatrixError {
	if r == nil {
		return nil
	}

	var matrixErr *MatrixError
	data, _ := io.ReadAll(io.LimitReader(r, maxMatrixErrorBody)) //nolint:errcheck // ignore error as we will return nil
	if err := json.Unmarshal(data, &matrixErr); err != nil {
		return NewMatrixError("M_UNKNOWN", fmt.Sprintf("failed to decode error response %q: %v", string(data), err))
	}

	return matrixErr
}

// IsContextError reports whether err is a context cancellation or a blown deadline, the two you usually treat alike.
func IsContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
