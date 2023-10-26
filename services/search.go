package services

import (
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"

	"gitlab.com/etke.cc/mrs/api/config"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

// Search service
type Search struct {
	cfg       *config.Search
	stub      []*model.Entry
	repo      SearchRepository
	block     BlocklistService
	stopwords map[string]struct{}
}

// SearchRepository interface
type SearchRepository interface {
	Search(searchQuery query.Query, limit, offset int, sortBy []string) ([]*model.Entry, int, error)
}

// SearchFieldsBoost field name => boost
var SearchFieldsBoost = map[string]float64{
	"language": 100,
	"name":     10,
	"server":   10,
	"alias":    5,
}

// NewSearch creates new search service
func NewSearch(cfg *config.Search, repo SearchRepository, block BlocklistService, stoplist []string) *Search {
	stopwords := make(map[string]struct{}, len(stoplist))
	for _, stopword := range stoplist {
		stopwords[stopword] = struct{}{}
	}

	s := &Search{
		cfg:       cfg,
		repo:      repo,
		block:     block,
		stopwords: stopwords,
	}
	s.initStubs()

	return s
}

// initStubs prepares stub rooms from config
func (s *Search) initStubs() {
	s.stub = make([]*model.Entry, 0, len(s.cfg.EmptyResults))
	for _, stub := range s.cfg.EmptyResults {
		s.stub = append(s.stub, &model.Entry{
			ID:        stub.ID,
			Type:      "room",
			Alias:     stub.Alias,
			Name:      stub.Name,
			Topic:     stub.Topic,
			Avatar:    stub.Avatar,
			Server:    stub.Server,
			Members:   stub.Members,
			Language:  stub.Language,
			AvatarURL: stub.AvatarURL,
		})
	}
}

// emptyResults returned when no query is provided
func (s *Search) emptyResults() ([]*model.Entry, int, error) {
	return s.stub, len(s.stub), nil
}

// Search things
// ref: https://blevesearch.com/docs/Query-String-Query/
func (s *Search) Search(query, sortBy string, limit, offset int) ([]*model.Entry, int, error) {
	if query == "" {
		return s.emptyResults()
	}
	if limit == 0 {
		limit = s.cfg.Defaults.Limit
	}
	if offset == 0 {
		offset = s.cfg.Defaults.Offset
	}

	builtQuery := s.getSearchQuery(s.matchFields(query))
	if builtQuery == nil {
		return []*model.Entry{}, 0, nil
	}
	results, total, err := s.repo.Search(builtQuery, limit, offset, utils.StringToSlice(sortBy, s.cfg.Defaults.SortBy))
	if err != nil {
		return nil, 0, err
	}

	return s.removeBlocked(results), total, nil
}

// removeBlocked removes results from blocked servers from the search results
func (s *Search) removeBlocked(results []*model.Entry) []*model.Entry {
	allowed := []*model.Entry{}
	for _, entry := range results {
		if entry.IsBlocked(s.block) {
			continue
		}

		allowed = append(allowed, entry)
	}

	return allowed
}

func (s *Search) matchFields(query string) (string, map[string]string) {
	if !strings.Contains(query, ":") { // if no key:value pair(-s) - nothing is here
		return query, nil
	}
	fields := map[string]string{}
	parts := strings.Split(query, " ") // e.g. "language:EN foss"
	toRemove := []string{}
	for _, part := range parts {
		if !strings.Contains(part, ":") { // no key:value pair
			continue
		}
		toRemove = append(toRemove, part)

		pair := strings.Split(strings.TrimSpace(part), ":")
		if len(pair) < 2 {
			continue
		}
		fields[strings.TrimSpace(pair[0])] = strings.TrimSpace(pair[1])
	}

	for _, remove := range toRemove {
		query = strings.ReplaceAll(query, remove, "")
	}
	query = strings.TrimSpace(query)

	return query, fields
}

type bleveQuery interface {
	query.Query
	SetField(string)
	SetBoost(float64)
}

func (s *Search) newMatchQuery(match, field string, phrase bool) bleveQuery {
	var searchQuery bleveQuery
	if phrase {
		searchQuery = bleve.NewMatchPhraseQuery(match)
	} else {
		searchQuery = bleve.NewMatchQuery(match)
	}
	searchQuery.SetField(field)
	searchQuery.SetBoost(SearchFieldsBoost[field])

	return searchQuery
}

// shouldReject checks if query or fields contain words from the stoplist
func (s *Search) shouldReject(q string, fields map[string]string) bool {
	for k, v := range fields {
		if _, ok := s.stopwords[k]; ok {
			return true
		}
		if _, ok := s.stopwords[v]; ok {
			return true
		}
	}
	for _, k := range strings.Split(q, " ") {
		if _, ok := s.stopwords[k]; ok {
			return true
		}
	}
	return false
}

func (s *Search) getSearchQuery(q string, fields map[string]string) query.Query {
	// base/standard query
	q = strings.TrimSpace(q)
	if s.shouldReject(q, fields) {
		return nil
	}

	phrase := strings.Contains(q, " ")
	queries := []query.Query{
		s.newMatchQuery(q, "name", phrase),
		s.newMatchQuery(q, "alias", phrase),
		s.newMatchQuery(q, "topic", phrase),
		s.newMatchQuery(q, "server", phrase),
	}

	// optional fields, like "language:EN"
	if len(fields) > 0 {
		boolQ := bleve.NewBooleanQuery()
		for field, fieldQ := range fields {
			boolQ.AddMust(s.newMatchQuery(fieldQ, field, false))
		}
		queries = append(queries, boolQ)
	}

	return bleve.NewDisjunctionQuery(queries...)
}
