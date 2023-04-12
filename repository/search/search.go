package search

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"

	"gitlab.com/etke.cc/mrs/api/model"
)

// Search something!
// ref: https://blevesearch.com/docs/Query-String-Query/
func (i *Index) Search(query string, limit, offset int) ([]model.Entry, error) {
	bleveQuery := bleve.NewQueryStringQuery(query)
	req := bleve.NewSearchRequestOptions(bleveQuery, limit, offset, false)
	req.Fields = []string{"*"}

	resp, err := i.index.Search(req)
	if err != nil {
		return nil, err
	}
	if resp.Total == 0 {
		return nil, nil
	}

	return parseSearchResults(resp.Hits), nil
}

func parseSearchResults(result []*search.DocumentMatch) []model.Entry {
	entries := make([]model.Entry, 0, len(result))
	for _, hit := range result {
		entries = append(entries, model.Entry{
			ID:      hit.ID,
			Type:    parseHitField[string](hit, "type"),
			Alias:   parseHitField[string](hit, "alias"),
			Name:    parseHitField[string](hit, "name"),
			Topic:   parseHitField[string](hit, "topic"),
			Avatar:  parseHitField[string](hit, "avatar"),
			Server:  parseHitField[string](hit, "server"),
			Members: int(parseHitField[float64](hit, "members")),
		})
	}

	return entries
}

func parseHitField[T any](hit *search.DocumentMatch, field string) T {
	var zero T
	v, ok := hit.Fields[field].(T)
	if !ok {
		return zero
	}

	return v
}
