# workpool

[![Go Reference](https://pkg.go.dev/badge/github.com/etkecc/go-kit.svg)](https://pkg.go.dev/github.com/etkecc/go-kit/workpool)

A bounded goroutine pool. You want to do 10,000 things concurrently but not with 10,000 goroutines, so you hand them to a fixed crew of workers who chew through the pile. Queue the work, call `Run`, wait for it to drain.

Part of the dependency-free root module.

```go
go get github.com/etkecc/go-kit
```

```go
import "github.com/etkecc/go-kit/workpool"

wp := workpool.New(8) // 8 workers, started immediately
for _, item := range items {
    item := item
    wp.Do(func() { process(item) }) // chainable, blocks if the queue is full
}
wp.Run() // blocks until every queued task is done, then closes the pool
```

Workers start the moment you call `New`, not when you call `Run`. `Do` is chainable and blocks if the queue is full (buffer defaults to `workers*100+1`). `Run` waits for the queue to drain, then closes the pool.

## The missing guardrail: `Do` after `Run` vanishes

There is exactly one way to get hurt here, so here's the sign nailed to the tree at the edge of the cliff. Once you've called `Run`, the pool is closed, and any `Do` after that is a **silent no-op**. No error. No panic. The task doesn't run, doesn't queue, doesn't warn you: it's just gone, swallowed by a pool that isn't listening anymore. You find out when the results are short and nothing in the logs says why.

So the rule is simple: **add all your work before you call `Run`.** The pool is single-use, there's no reset, no second act. If you have another batch, build another pool.

Two more small things worth knowing: a panic inside a task is recovered and logged so one bad task can't take the pool down, but it's logged, not handed back to you. And `Run` busy-waits while it drains rather than sleeping, so it burns a little CPU on the way out; fine for batch work, not what you want on a latency-sensitive shutdown path.

The buffer-sizing knob and the rest are in [godoc](https://pkg.go.dev/github.com/etkecc/go-kit/workpool).

## License

GNU LGPL-3.0. See [../LICENSE](../LICENSE).
