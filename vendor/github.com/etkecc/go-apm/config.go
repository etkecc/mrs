package apm

import (
	"runtime/debug"
	"sync"

	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

var (
	hc            Healthchecks
	loglevel      zerolog.Level
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

// SetLogLevel sets the log level
func SetLogLevel(level string) {
	var err error
	loglevel, err = zerolog.ParseLevel(level)
	if err != nil {
		loglevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(loglevel)
}

// SetSentryDSN sets the sentry DSN and initializes it
func SetSentryDSN(dsn string) {
	sentryDSN = dsn
	initSentry()
}

// SetHealthchecks sets the healthchecks client
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
		NewLoggerPlain().Error().Err(err).Msg("sentry initialization failed")
		return
	}

	initialized = true
}
