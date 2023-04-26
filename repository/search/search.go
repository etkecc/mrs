package search

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"

	"gitlab.com/etke.cc/mrs/api/model"
)

// Search something!
func (i *Index) Search(searchQuery query.Query, limit, offset int, sortBy []string) ([]*model.Entry, error) {
	req := bleve.NewSearchRequestOptions(searchQuery, limit, offset, false)
	req.Fields = []string{"*"}
	req.SortBy(sortBy)

	resp, err := i.index.Search(req)
	if err != nil {
		return nil, err
	}
	if resp.Total == 0 {
		return nil, nil
	}

	return parseSearchResults(resp.Hits), nil
}

func parseSearchResults(result []*search.DocumentMatch) []*model.Entry {
	entries := make([]*model.Entry, 0, len(result))
	for _, hit := range result {
		entries = append(entries, &model.Entry{
			ID:        hit.ID,
			Type:      parseHitField[string](hit, "type"),
			Alias:     parseHitField[string](hit, "alias"),
			Name:      parseHitField[string](hit, "name"),
			Topic:     parseHitField[string](hit, "topic"),
			Avatar:    parseHitField[string](hit, "avatar"),
			AvatarURL: parseHitField[string](hit, "avatar_url"),
			Server:    parseHitField[string](hit, "server"),
			Language:  parseHitField[string](hit, "language"),
			Members:   int(parseHitField[float64](hit, "members")),
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
