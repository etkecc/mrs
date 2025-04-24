package apm

import (
	"context"
	"fmt"
	"os"

	sentryzerolog "github.com/getsentry/sentry-go/zerolog"
	"github.com/rs/zerolog"
)

// Log returns a logger with the context provided, if no context is provided, it will return a logger with a new context
func Log(ctx ...context.Context) *zerolog.Logger {
	if len(ctx) > 0 {
		return zerolog.Ctx(ctx[0])
	}
	return zerolog.Ctx(NewContext())
}

// NewLogger returns a new logger with sentry integration (if possible)
func NewLogger(ctx context.Context) *zerolog.Logger {
	var w zerolog.LevelWriter

	consoleWriter := zerolog.LevelWriterAdapter{
		Writer: zerolog.ConsoleWriter{
			Out:          os.Stdout,
			PartsExclude: []string{zerolog.TimestampFieldName},
		},
	}

	sentryWriter, err := newSentryWriter(ctx)
	if err == nil {
		w = zerolog.MultiLevelWriter(sentryWriter, consoleWriter)
	} else {
		w = consoleWriter
	}

	log := zerolog.New(w).With().Timestamp().Caller().Logger().Hook(hcHook)
	return &log
}

// NewLoggerPlain returns a new logger without sentry integration
func NewLoggerPlain() *zerolog.Logger {
	consoleWriter := zerolog.ConsoleWriter{
		Out:          os.Stdout,
		PartsExclude: []string{zerolog.TimestampFieldName},
	}

	log := zerolog.New(consoleWriter)
	return &log
}

func newSentryWriter(ctx context.Context) (zerolog.LevelWriter, error) {
	if sentryDSN == "" {
		return nil, fmt.Errorf("sentry DSN not set")
	}

	return sentryzerolog.NewWithHub(GetHub(ctx), sentryzerolog.Options{
		Levels:          []zerolog.Level{zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel},
		WithBreadcrumbs: true,
	})
}
