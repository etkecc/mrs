# Matrix Rooms Search

A fully-featured, standalone, matrix rooms search service.

## How it works?

1. Discover matrix servers (a.k.a find alive and properly configured) from provided config
2. Parse public rooms from the discovered servers
3. Ingest parsed public rooms into search index

Each step can be run separately or all at once using admin API

## API

Check [openapi.yml](./openapi.yml)

## todo

* indexed entry's `members` is always `0`, should be fixed by proper index mapping
