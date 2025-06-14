# Development

## How to add a new field

1. Adjust `model/matrix.go` and `model/search.go` if needed
2. Adjust `repository/search/bleve.go` `getIndexMapping()`
3. Adjust `repository/search/search.go` `parseSearchResults()`
4. Adjust `services/search.go` `getSearchQuery()`
