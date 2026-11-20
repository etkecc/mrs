# APM

Wrapper around [Sentry](https://sentry.io) and [log/slog](https://pkg.go.dev/log/slog) to provide a simple way to instrument your Go applications.

## Features

* Painless transactions and spans - `apm.StartSpan()`
* Context-aware logging and error handling - `apm.NewContext()`, `apm.NewLogger()`
* Automatic HTTP client instrumentation - `apm.WrapClient()`, `apm.WrapRoundTripper()`
* Echo v5 middleware - `apm.WithSentry()`

## Usage

```go
package main

// configure APM on your app initialization
apm.SetName("my-app")
apm.SetLogLevel("debug")
apm.SetSentryDSN("https://sentry.io/your-dsn")

// optionally, if you have gitlab.com/etke.cc/go/healthchecks client
apm.SetHealthchecks(healthchecksClient)

// thats it! you are ready to go

// create a new context with correct sentry hub and logger
ctx := apm.NewContext()

// or instrument an existing context
ctx = apm.NewContext(ctx)

// log through the context-aware slog logger
apm.Log(ctx).Info("request handled", "user", "alice")

// wrap your http client
client := apm.WrapClient(nil) // wraps http.DefaultClient when no client is provided

// automatic request instrumentation and retries for 5xx errors
client.Do(req)

// work with transactions and spans. No transaction in context? Will be created automatically!
span := apm.StartSpan(ctx, "my-span")
```

### Echo

```go
e := echo.New()
e.Logger = apm.NewLoggerPlain()
e.Use(apm.WithSentry())
```

Error-level records logged through `apm.NewLogger()` are reported to Sentry and to the configured healthchecks client.

## Data sent to Sentry

* Error-level records become Sentry events; WARN, INFO, and DEBUG records become breadcrumbs that ship with the next event, including their attributes.
* Attributes become event tags, `error`/`err` becomes the event exception, and `user`, `fingerprint`, and `transaction` fill the matching event fields.
* `apm.WithSentry()` attaches the request to events captured during that request. Sentry's default data collection sends the URL, query, and headers (minus its built-in denylist), and omits cookies, request bodies, and client IPs.
* `apm.NewContext(parent)` reuses a Sentry hub found in `parent`. Start each request from a hub-free context so it gets its own clone.

Call `apm.SetName`, `apm.SetSentryDSN`, and `apm.SetHealthchecks` during startup.

## Install

```sh
go get github.com/etkecc/go-apm/v2
```

The v2 module targets Echo v5 and `log/slog`.
