package utils

import (
	"io"
	"os"
	"time"

	zlogsentry "github.com/archdx/zerolog-sentry"
	"github.com/rs/zerolog"
	"github.com/xxjwxc/gowp/workpool"
)

// Logger default
var Logger = SetupLogger("", "info")

func SetupLogger(level, sentryDSN string) *zerolog.Logger {
	loglevel, err := zerolog.ParseLevel(level)
	if err != nil {
		loglevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(loglevel)
	var w io.Writer
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, PartsExclude: []string{zerolog.TimestampFieldName}}
	sentryWriter, err := zlogsentry.New(sentryDSN, zlogsentry.WithBreadcrumbs())
	if err == nil {
		w = io.MultiWriter(sentryWriter, consoleWriter)
	} else {
		w = consoleWriter
	}
	log := zerolog.New(w).With().Timestamp().Logger()

	return &log
}

func PoolProgress(wp *workpool.WorkPool, callback func(), optionalDuration ...time.Duration) {
	every := 1 * time.Minute
	if len(optionalDuration) > 0 {
		every = optionalDuration[0]
	}

	for {
		if wp.IsDone() {
			return
		}
		callback()
		time.Sleep(every)
	}
}
