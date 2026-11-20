package apm

import (
	"fmt"
	"log/slog"
	"math"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/getsentry/sentry-go"
)

var (
	hc            Healthchecks
	loglevel      slog.LevelVar
	initialized   bool
	initMu        sync.Mutex
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

// SetLogLevel sets the level of loggers created by this package, defaulting to info on unknown names
func SetLogLevel(level string) {
	parsed, err := parseLogLevel(level)
	if err != nil {
		parsed = slog.LevelInfo
	}
	loglevel.Set(parsed)
}

// parseLogLevel maps a level name, including zerolog-era names, to a slog level
func parseLogLevel(level string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace":
		return slog.LevelDebug - 4, nil
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	case "fatal":
		return slog.LevelError + 4, nil
	case "panic":
		return slog.LevelError + 8, nil
	case "disabled":
		return slog.Level(math.MaxInt), nil
	default:
		return 0, fmt.Errorf("unknown log level %q", level)
	}
}

// SetSentryDSN sets the sentry DSN and initializes it; call it during startup
func SetSentryDSN(dsn string) {
	sentryDSN = dsn
	initSentry()
}

// SetHealthchecks sets the healthchecks client; call it during startup
func SetHealthchecks(h Healthchecks) {
	hc = h
}

// initSentry initializes sentry
func initSentry() {
	initMu.Lock()
	defer initMu.Unlock()

	if initialized {
		return
	}

	if sentryDSN == "" {
		return
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:         sentryDSN,
		Environment: sentryName,
		Release:     sentryVersion,
	})
	if err != nil {
		NewLoggerPlain().Error("sentry initialization failed", "error", err)
		return
	}

	initialized = true
}
