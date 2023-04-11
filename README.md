# Matrix Rooms Search

A fully-featured, standalone, matrix rooms search service.

## How it works?

1. Discover matrix servers (a.k.a find alive and properly configured) from provided config
2. Parse public rooms from the discovered servers
3. Ingest parsed public rooms into search index

Each step can be run separately or all at once using admin API

### API

Check [openapi.yml](./openapi.yml)

## Quick Start

1. Build mrs
2. Copy `config.yml.sample` into `config.yml` and adjust it
3. Run mrs with `-c config.yml`
4. You probably want to call `/-/full` admin API endpoint at start

## Development

### how to add new field

1. adjust `model/matrix.go` and `model/search.go` if needed
2. adjust `repository/search/bleve.go` `getIndexMapping()`
3. adjust `repository/search/search.go` `parseSearchResults()`
