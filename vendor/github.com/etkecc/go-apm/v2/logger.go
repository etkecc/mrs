package apm

import (
	"context"
	"log/slog"
	"os"

	"github.com/getsentry/sentry-go"
)

var silentLogger = slog.New(slog.DiscardHandler)

// loggerKey is the context key used to store the logger created by NewContext
type loggerKey struct{}

// Log returns the logger stored in the context, a fresh logger without a context, or a discarding logger
func Log(ctx ...context.Context) *slog.Logger {
	if len(ctx) == 0 {
		return Log(NewContext())
	}
	if ctx[0] != nil {
		if logger, ok := ctx[0].Value(loggerKey{}).(*slog.Logger); ok {
			return logger
		}
	}
	return silentLogger
}

// NewLogger returns a new logger that reports error-level records to sentry and healthchecks
func NewLogger(ctx context.Context) *slog.Logger {
	var hub *sentry.Hub
	if sentryDSN != "" {
		hub = GetHub(ctx)
	}
	return slog.New(&reportHandler{
		next: slog.NewTextHandler(os.Stdout, logOptions(true)),
		hub:  hub,
	})
}

// NewLoggerPlain returns a new logger without sentry and healthchecks integration
func NewLoggerPlain() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, logOptions(false)))
}

// logOptions returns handler options honoring the package log level without timestamps
func logOptions(source bool) *slog.HandlerOptions {
	return &slog.HandlerOptions{AddSource: source, Level: &loglevel, ReplaceAttr: dropTime}
}

// dropTime removes the built-in timestamp attribute
func dropTime(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) == 0 && attr.Key == slog.TimeKey && attr.Value.Kind() == slog.KindTime {
		return slog.Attr{}
	}
	return attr
}
