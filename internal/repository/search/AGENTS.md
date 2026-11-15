# internal/repository/search

Bleve index for room entries: mapping, analyzers, and index lifecycle; query construction stays in `internal/services/search.go`.

## Contract

- `name` and `topic` are stemmed by the `multilang` analyzer; `name_exact` and `topic_exact` are indexed-only, unstemmed twins that catch a full word the stemmer clipped.
- `SearchFieldsBoost` and the exact-field loop in `internal/services/search.go` name these fields, so the mapping field names are a contract across both packages.

## Landmines

- `getIndexMapping` applies only when `bleve.New` runs: an existing index opens with the mapping it was built with, so a mapping change needs a fresh index (`Swap` or removing the path).
- `multilang.Register` is process-global and first-writer-wins: the detector and default language from the first `NewIndex` govern later indexes, whose registration errors are logged and ignored.
- `Swap` recovers bleve scorch's data-race panic and then returns nil, leaving the index closed if the reopen did not run.

## Verify

`just test ./internal/repository/search`

## Pointers

- `internal/services/search.go` - query builder and field boosts that must match this package's mapping.
- Root rules: `../../../AGENTS.md`.
