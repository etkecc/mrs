# internal/services

Service layer for MRS: `cmd/mrs` constructs it and the cron pipeline drives `DataFacade`; storage and federation sit behind interfaces.

## Contract

- `DataFacade.Full` runs discovery, parsing, then `Ingest`.
- `Index.RoomsBatch` accumulates until `batch.rooms`, then flushes; callers adding rooms outside `DataFacade.Ingest` must call `IndexBatch` to flush the tail.
- `Validator.IsRoomAllowed` requires room ID and alias and drops blocked IDs, aliases, servers, and `<matrix.server_name>` noindex topics.
- `Crawler.DiscoverServers`, `ParseRooms`, and `EachRoom` are single-flight; a concurrent call logs and returns.
- `Crawler.loadServers` dials online and never-checked rows every run; offline rows wait on a backoff keyed by `OnlineAt` and `CheckedAt`.

## Landmines

- `Crawler.afterRoomParsing` prunes rooms not parsed in 7 days and offline servers older than 30 days.
- `Search` builds its stopword set once in `NewSearch`; changing `blocklist.queries` needs a restart.
- `DataFacade.Ingest` empties the live index before repopulating it; a failure after the swap leaves search empty until a successful reindex.

## Verify

`just test ./internal/services`

## Pointers

- `internal/services/matrix/AGENTS.md` - Matrix federation and key verification. Read before touching matrix.
- `../../AGENTS.md` - root commands, generated files, and lint rules.
