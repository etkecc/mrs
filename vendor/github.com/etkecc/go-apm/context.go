package apm

import (
	"context"

	"github.com/getsentry/sentry-go"
)

// NewContext creates a new context with a logger and sentry hub
func NewContext(parent ...context.Context) context.Context {
	ctx := context.Background()
	if len(parent) > 0 {
		ctx = parent[0]
	}

	hub := GetHub(ctx)
	if hub == nil && sentryDSN != "" {
		hub = sentry.CurrentHub().Clone()
		ctx = sentry.SetHubOnContext(ctx, hub)
	}
	return NewLogger(ctx).WithContext(ctx)
}
