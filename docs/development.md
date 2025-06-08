# Development

## how to add new field

1. adjust `model/matrix.go` and `model/search.go` if needed
2. adjust `repository/search/bleve.go` `getIndexMapping()`
3. adjust `repository/search/search.go` `parseSearchResults()`
4. adjust `services/search.go` `getSearchQuery()`
