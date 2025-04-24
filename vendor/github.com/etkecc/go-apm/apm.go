package apm

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

// skipStackPgs is a list of packages to skip in the stack trace
var skipStackPgs = []string{
	"github.com/etkecc/go-apm",
	"github.com/rs/zerolog",
}

// Error captures the error and sends it to sentry and healthchecks
func Error(ctx context.Context, err error) {
	if err == nil {
		return
	}

	GetHub(ctx).CaptureException(err)
	HealthcheckFail(strings.NewReader("error: " + err.Error()))
}

// StartSpan starts a new span, and if there is no transaction, it starts a new transaction
func StartSpan(ctx context.Context, operation string) *sentry.Span {
	if transaction := sentry.TransactionFromContext(ctx); transaction == nil {
		ctx = sentry.StartTransaction(ctx, operation, sentry.WithDescription(operation)).Context()
	}
	return sentry.StartSpan(ctx, operation, sentry.WithDescription(operation))
}

// GetHub returns the hub from the context (if context is provided and has a hub) or the current hub
func GetHub(ctx ...context.Context) *sentry.Hub {
	if len(ctx) == 0 {
		return sentry.CurrentHub().Clone()
	}

	if hub := sentry.GetHubFromContext(ctx[0]); hub != nil {
		return hub
	}

	return sentry.CurrentHub().Clone()
}

// Flush sends the events to sentry
func Flush(ctx ...context.Context) {
	GetHub(ctx...).Flush(5 * time.Second)
}

// Recover sends the error to sentry
func Recover(err any, repanic bool, ctx ...context.Context) {
	if err == nil {
		Flush(ctx...)
		return
	}
	HealthcheckFail(strings.NewReader(fmt.Sprintf("panic recovered: %+v", err)))
	if len(ctx) > 0 && ctx[0] != nil {
		GetHub(ctx...).RecoverWithContext(ctx[0], err)
	} else {
		GetHub(ctx...).Recover(err)
	}
	Flush(ctx...)
	if repanic {
		panic(err)
	}
}

// GetStackTrace returns the stack trace, skipping the apm and logger packages
//
//nolint:gocognit // this function is complex by nature
func GetStackTrace() string {
	// create stack trace
	var stack string
	buf := make([]byte, 1<<16) // 64kb by default
	for {                      // loop until we get the full stack trace
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			buf = buf[:n] // Trim to actual size
			stack = string(buf)
			break
		}
		// Double the buffer size if it wasn't large enough
		buf = make([]byte, len(buf)*2)
	}

	// Split stack trace into lines
	stackLines := strings.Split(stack, "\n")
	trimmedStackLines := make([]string, 0)

	skipNextLine := false
	for _, line := range stackLines {
		if strings.TrimSpace(line) == "" {
			continue // skip empty lines
		}

		if skipNextLine {
			skipNextLine = false
			continue
		}

		// If this is a function call line (not a source line)
		if !strings.HasPrefix(line, "\t") {
			skip := false
			for _, pkg := range skipStackPgs {
				if strings.HasPrefix(line, pkg) {
					skip = true
					break
				}
			}
			skipNextLine = skip
		}

		if !skipNextLine {
			trimmedStackLines = append(trimmedStackLines, line)
		}
	}

	// Join the trimmed lines back to a single string
	return strings.Join(trimmedStackLines, "\n")
}
