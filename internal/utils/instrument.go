package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	zlogsentry "github.com/archdx/zerolog-sentry"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

var (
	loglevel      zerolog.Level
	sentryDSN     string
	sentryName    string
	sentryVersion = func() string {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					return setting.Value
				}
			}
		}
		return "development"
	}()
)

// SetName sets the name of the application
func SetName(name string) {
	sentryName = name
}

// SetLogLevel sets the log level
func SetLogLevel(level string) {
	var err error
	loglevel, err = zerolog.ParseLevel(level)
	if err != nil {
		loglevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(loglevel)
}

// SetSentryDSN sets the sentry DSN
func SetSentryDSN(dsn string) {
	sentryDSN = dsn
}

// NewContext creates a new context with a logger and sentry hub
func NewContext(parent ...context.Context) context.Context {
	ctx := context.Background()
	if len(parent) > 0 {
		ctx = parent[0]
	}

	hub := sentry.GetHubFromContext(ctx)
	if hub == nil && sentryDSN != "" {
		hub = sentry.CurrentHub().Clone()
		ctx = sentry.SetHubOnContext(ctx, hub)
	}
	return newLogger(ctx).WithContext(ctx)
}

// StartSpan starts a new span, and if there is no transaction, it starts a new transaction
func StartSpan(ctx context.Context, operation string) *sentry.Span {
	if transaction := sentry.TransactionFromContext(ctx); transaction == nil {
		ctx = sentry.StartTransaction(ctx, operation, sentry.WithDescription(operation)).Context()
	}
	return sentry.StartSpan(ctx, operation, sentry.WithDescription(operation))
}

func newSentryWriter(ctx context.Context) (io.Writer, error) {
	if sentryDSN == "" {
		return nil, fmt.Errorf("sentry DSN not set")
	}

	if hub := sentry.GetHubFromContext(ctx); hub != nil && hub.Scope() != nil && hub.Client() != nil {
		return zlogsentry.NewWithHub(hub, getSentryOptions()...)
	}
	return zlogsentry.New(sentryDSN, getSentryOptions()...)
}

func newLogger(ctx context.Context) zerolog.Logger {
	var w io.Writer
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, PartsExclude: []string{zerolog.TimestampFieldName}}
	sentryWriter, err := newSentryWriter(ctx)
	if err == nil {
		w = io.MultiWriter(sentryWriter, consoleWriter)
	} else {
		w = consoleWriter
	}
	return zerolog.New(w).With().Timestamp().Caller().Logger()
}

func getSentryOptions() []zlogsentry.WriterOption {
	return []zlogsentry.WriterOption{
		zlogsentry.WithBreadcrumbs(),
		zlogsentry.WithTracing(),
		zlogsentry.WithSampleRate(0.25),
		zlogsentry.WithTracingSampleRate(0.25),
		zlogsentry.WithRelease(sentryName + "@" + sentryVersion),
	}
}
