# httpclient

Tuned `*http.Client` constructors with safe-replay retries and an optional SSRF dial guard; root module, depends on `retry`.

## Contract

- Five constructors: `New`, `NewSingleHost`, `NewMultiHost`, `Wrap`, `NewTransport`; construction returns no error.
- `TransportOption` and `RetryOption` are split at compile time; the full constructors accept either.
- Defaults: 3 total attempts, 200ms linear step, 10s per-attempt timeout, TLS 1.2 floor; client `Timeout` stays 0.
- Retries 429/500/502/503/504 and safe methods by default; POST/PATCH need `WithRetryNonIdempotent(true)`.
- `WithDialGuard` checks the resolved IP at dial time and refuses loopback, private, link-local, and metadata ranges.
- `NewTransport` returns the bare transport; retry lives in the other constructors.

## Landmines

- A retryable request with a body and no `GetBody` returns `ErrNonReplayableBody`; provide `GetBody` for replays.
- `WithDialGuard` nulls an egress proxy because the guard would only see the proxy IP.
- `WithDialContext` replaces the built-in dialer, so a custom dialer must enforce the SSRF policy itself.
- `Wrap` on a client already carrying this retry layer multiplies attempts; wrap the base client once.
- `WithRetry` replaces the default retrier, so `WithRetryIf`, `WithMaxRetries`, and `WithRetryDelayStep` are ignored.
- `WithOnAttempt` fires concurrently; keep the hook concurrency-safe.

## Verify

`go test ./httpclient`

## Pointers

- `README.md` - constructor presets, type split, SSRF guard, retry rules.
- Root rules: `../AGENTS.md`.
