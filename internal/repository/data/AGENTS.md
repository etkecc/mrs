# internal/repository/data

bbolt store for servers, rooms, moderation lists, and index stats; it owns the bucket schema and the server lifecycle clocks.

## Contract

- `rooms_mappings` is one bucket holding both directions (`roomID -> alias` and `alias -> roomID`); `AddRoomMapping`, `RemoveRoomMapping`, and `RecreateRoomMapping` keep both in one transaction.
- `MatrixServer.OnlineAt` anchors the 1-month prune and the offline backoff base; `CheckedAt` is the last dial attempt.

## Landmines

- `initBuckets` deletes every top-level bucket not in `buckets` on each `New`: a new bucket constant missing from that list loses its data on the next start, and another schema must not share the file.
- `AddRoomBatch` only buffers (10000 rooms per flush); `Close` does not flush. Call `FlushRoomBatch` at the end of a parse run, or the tail never reaches bbolt.
- Marking a server offline must not move `OnlineAt`: the prune reads it, so dead servers never expire; a stub left at zero `OnlineAt` is pruned on sight.
- `EachRoom` skips rooms in `rooms_banlist` and decodes into one reused `*model.MatrixRoom`: a handler that stores the pointer ends up with the last room.

## Verify

`just test ./internal/repository/data`

## Pointers

- `internal/services/crawler.go` - end-of-parse flush, 1-month server prune, and mapping rebuild. Read before changing room batching or server lifecycle writes.
- Root rules: `../../../AGENTS.md`.
