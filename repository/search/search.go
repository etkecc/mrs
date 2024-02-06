package search

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"

	"gitlab.com/etke.cc/mrs/api/model"
)

// Search something!
func (i *Index) Search(searchQuery query.Query, limit, offset int, sortBy []string) (results []*model.Entry, total int, err error) {
	req := bleve.NewSearchRequestOptions(searchQuery, limit, offset, false)
	req.Fields = []string{"*"}
	req.SortBy(sortBy)

	resp, err := i.index.Search(req)
	if err != nil {
		return nil, 0, err
	}
	if resp.Total == 0 {
		return nil, 0, nil
	}

	return parseSearchResults(resp.Hits), int(resp.Total), nil
}

func parseSearchResults(result []*search.DocumentMatch) []*model.Entry {
	entries := make([]*model.Entry, 0, len(result))
	for _, hit := range result {
		entries = append(entries, &model.Entry{
			ID:            hit.ID,
			Type:          parseHitField[string](hit, "type"),
			Alias:         parseHitField[string](hit, "alias"),
			Name:          parseHitField[string](hit, "name"),
			Topic:         parseHitField[string](hit, "topic"),
			Avatar:        parseHitField[string](hit, "avatar"),
			Server:        parseHitField[string](hit, "server"),
			Members:       int(parseHitField[float64](hit, "members")),
			Language:      parseHitField[string](hit, "language"),
			AvatarURL:     parseHitField[string](hit, "avatar_url"),
			RoomType:      parseHitField[string](hit, "room_type"),
			JoinRule:      parseHitField[string](hit, "join_rule"),
			GuestJoinable: parseHitField[bool](hit, "guest_can_join"),
			WorldReadable: parseHitField[bool](hit, "world_readable"),
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
