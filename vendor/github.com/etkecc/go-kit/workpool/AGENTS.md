# workpool

Bounded goroutine pool in the root module; fixed workers start in `New`, `Do` queues tasks, `Run` drains and closes.

## Contract

- `New(workers, optionalBufferSize...)` clamps workers to at least 1; the queue buffer defaults to `workers*100+1`.
- `Do` is chainable and blocks when the queue is full; it returns the receiver for chaining.
- `Run` blocks until every queued task finishes, then closes the queue; a second concurrent `Run` is a no-op.
- `IsRunning` reports whether any task is queued or in flight.
- Task panics are recovered and printed to stdout; the caller sees no error.

## Landmines

- Queue all work before `Run`; `Do` after `Run` silently drops the task because the pool is single-use.
- Task panics are only printed, so a failed task does not fail `Run`; check results out of band.
- The pool is single-use; build a new pool for another batch.

## Verify

`go test -race ./workpool`

## Pointers

- `README.md` - queue sizing, single-use behavior, panic handling.
- Root rules: `../AGENTS.md`.
