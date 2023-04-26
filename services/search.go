package services

import (
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"

	"gitlab.com/etke.cc/mrs/api/model"
)

// Search service
type Search struct {
	repo SearchRepository
}

// SearchRepository interface
type SearchRepository interface {
	Search(searchQuery query.Query, limit, offset int, sortBy []string) ([]*model.Entry, error)
}

// NewSearch creates new search service
func NewSearch(repo SearchRepository) Search {
	return Search{repo}
}

// Search things
// ref: https://blevesearch.com/docs/Query-String-Query/
func (s Search) Search(query string, limit, offset int, sortBy []string) ([]*model.Entry, error) {
	return s.repo.Search(s.getSearchQuery(query), limit, offset, sortBy)
}

type bleveQuery interface {
	query.Query
	SetField(string)
	SetBoost(float64)
}

func (s Search) newMatchQuery(match, field string, boost float64, phrase bool) bleveQuery {
	var searchQuery bleveQuery
	if phrase {
		searchQuery = bleve.NewMatchPhraseQuery(match)
	} else {
		searchQuery = bleve.NewMatchQuery(match)
	}
	searchQuery.SetField(field)
	searchQuery.SetBoost(boost)

	return searchQuery
}

func (s Search) getSearchQuery(q string) query.Query {
	q = strings.TrimSpace(q)
	phrase := strings.Contains(q, " ")
	return bleve.NewDisjunctionQuery(
		s.newMatchQuery(q, "name", 10, phrase),
		s.newMatchQuery(q, "alias", 5, phrase),
		s.newMatchQuery(q, "topic", 0, phrase),
		s.newMatchQuery(q, "server", 0, phrase),
	)
}
