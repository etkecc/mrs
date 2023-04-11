# Matrix Rooms Search

mrs has 3 modes:
* parse rooms across matrix federation, using list of matrix servers provided in config
* index parsed data into [bleve](https://blevesearch.com/)
* search indexed data (http api)

On app start:
1. Index is created or opened
2. Background process of rooms parsing across available servers is started, storing data and automatically ingesting it into index

Later you can manipulate mrs using api.

## todo

* add pagination support to rooms parser
* make room parsing background process
* allow room parsing to be called from api
* add full reindex (read all rooms from data and ingest them) as background process with ability to be called from api
* indexed entry's `members` is always `0`, should be fixed by proper index mapping

### planned api

* GET /search?q=QUERY - search it! Query syntax: https://blevesearch.com/docs/Query-String-Query/
* POST /-/parse - start rooms parsing across federation and ingesting it into the index. If procedure is in progress, request will be ignored
* POST /-/reindex - starts process of indexing parsed data.


## why store data and index?

Parsed data is a static thing and it just... exists, you know. But index may be corrupted, or lost, because index itself is some kind of a state. So, by default all parsed info will be automatically ingested into the index file, but if for some reason it's unavailable or corrupted, you can always ingest preprocessed data into a new index.
