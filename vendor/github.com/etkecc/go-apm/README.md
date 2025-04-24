# APM

Wrapper around [Sentry](https://sentry.io) and [zerolog](https://github.com/rs/zerolog) to provide a simple way to instrument your Go applications.


## Features

* Painless transactions and spans - `apm.StartSpan()`
* Context-aware logging and error handling - `apm.NewContext()`, `apm.NewLogger()`
* Automatic HTTP client instrumentation - `apm.WrapClient()`, `apm.WrapRoundTripper()`

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
ctx = apm.NewContext(context.Background())

// wrap your http client
client := apm.WrapClient() // wraps http.DefaultClient when no client is provided

// automatic request instrumentation and retries for 5xx errors
client.Do(req)

// work with transactions and spans. No transaction in context? Will be created automatically!
span := apm.StartSpan(ctx, "my-span")
```
