# retry

Linear backoff with full jitter and `Retry-After` hints; root module, used by `httpclient` and direct callers.

## Contract

- Defaults: 3 total attempts, 1s delay step, jitter on, retry any non-nil error; `WithMaxRetries` below 1 clamps to 1.
- Delay for retry i is random in `[0, step*(i+1))`; the final attempt returns without a sleep.
- A positive `DelayHinter` replaces the linear step; with jitter, up to 10% is added to the hint.
- `DoCtx` checks ctx before each attempt and sleeps with ctx; `Do` uses `context.Background()`.
- Nil receiver, ctx, and fn return `ErrNilRetry`, `ErrNilContext`, and `ErrNilFn`.

## Landmines

- Implement `SuggestedRetryDelay` on a value receiver; a pointer receiver wrapped by value is missed by `errors.As`, and the hint silently falls back to linear backoff.
- A zero or negative hint is ignored; the linear backoff is used.
- `Do` has no cancellation; use `DoCtx` when the caller has a deadline.

## Verify

`go test ./retry`

## Pointers

- `README.md` - defaults, jitter recipe, DelayHinter example.
- Root rules: `../AGENTS.md`.
