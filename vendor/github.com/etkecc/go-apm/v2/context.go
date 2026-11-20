package apm

import (
	"context"

	"github.com/getsentry/sentry-go"
)

// NewContext creates a new context with a logger and sentry hub
func NewContext(parent ...context.Context) context.Context {
	ctx := context.Background()
	if len(parent) > 0 && parent[0] != nil {
		ctx = parent[0]
	}

	if sentry.GetHubFromContext(ctx) == nil && sentryDSN != "" {
		ctx = sentry.SetHubOnContext(ctx, sentry.CurrentHub().Clone())
	}
	return context.WithValue(ctx, loggerKey{}, NewLogger(ctx))
}
