package utils

import (
	"io"
	"os"

	zlogsentry "github.com/archdx/zerolog-sentry"
	"github.com/rs/zerolog"
)

// Logger default
var Logger *zerolog.Logger = SetupLogger("", "info")

func SetupLogger(level, sentryDSN string) *zerolog.Logger {
	loglevel, err := zerolog.ParseLevel(level)
	if err != nil {
		loglevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(loglevel)
	var w io.Writer
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, PartsExclude: []string{zerolog.TimestampFieldName}}
	sentryWriter, err := zlogsentry.New(sentryDSN)
	if err == nil {
		w = io.MultiWriter(sentryWriter, consoleWriter)
	} else {
		w = consoleWriter
	}
	log := zerolog.New(w).With().Timestamp().Logger()

	return &log
}
