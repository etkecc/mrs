# crontab

In-process five-field cron scheduler in the root module; services register startup jobs and stop it on shutdown.

## Contract

- `New` starts a ticker; schedules run in UTC unless `WithLocation`; `AddJob` returns a parse error or `ErrClosed`.
- A job fires at most once per absolute minute; a still-running job skips the next tick unless `WithOverlap`.
- `Shutdown` stops the ticker, waits for in-flight jobs bounded by ctx, and returns `ctx.Err()` on timeout; it is idempotent.
- Job panics are recovered and sent to stderr or `WithPanicHandler`; a panic in the handler is swallowed.

## Landmines

- `MustAddJob` panics on a bad spec and on `ErrClosed` after shutdown; use it only for compile-time constants at startup.
- With `WithLocation`, DST fall-back fires the same wall-clock minute twice; make jobs idempotent.
- Wall-clock steps (NTP, VM pause) can double-fire or skip a minute without a log; make jobs idempotent.
- When both dom and dow are restricted, a day matches either field; `1 0 1 * 1` fires on the 1st and every Monday.

## Verify

`go test ./crontab`

## Pointers

- `README.md` - schedule grammar, DST behavior, overlap, shutdown.
- Root rules: `../AGENTS.md`.
