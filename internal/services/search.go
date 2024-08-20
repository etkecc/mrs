package services

import (
	"context"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/rs/zerolog"
	"golang.org/x/exp/slices"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

// Search service
type Search struct {
	cfg   ConfigService
	data  searchDataRepository
	repo  SearchRepository
	stats StatsService
	block BlocklistService
}

type searchDataRepository interface {
	GetBiggestRooms(ctx context.Context, limit, offset int) []*model.MatrixRoom
}

// SearchRepository interface
type SearchRepository interface {
	Search(ctx context.Context, searchQuery query.Query, limit, offset int, sortBy []string) ([]*model.Entry, int, error)
}

type StatsService interface {
	Get() *model.IndexStats
}

// SearchFieldsBoost field name => boost
var SearchFieldsBoost = map[string]float64{
	"language": 100,
	"name":     10,
	"server":   10,
	"alias":    5,
}

// NewSearch creates new search service
func NewSearch(cfg ConfigService, data searchDataRepository, repo SearchRepository, block BlocklistService, stats StatsService) *Search {
	s := &Search{
		cfg:   cfg,
		data:  data,
		repo:  repo,
		stats: stats,
		block: block,
	}

	return s
}

// Search things
// ref: https://blevesearch.com/docs/Query-String-Query/
func (s *Search) Search(ctx context.Context, q, sortBy string, limit, offset int) ([]*model.Entry, int, error) {
	span := utils.StartSpan(ctx, "searchSvc.Search")
	defer span.Finish()
	log := zerolog.Ctx(span.Context())
	if limit == 0 {
		limit = s.cfg.Get().Search.Defaults.Limit
	}
	if offset == 0 {
		offset = s.cfg.Get().Search.Defaults.Offset
	}

	var builtQuery query.Query
	if q == "" {
		entries, length := s.getEmptyQueryResults(span.Context(), limit, offset)
		return entries, length, nil
	}
	builtQuery = s.getSearchQuery(s.matchFields(q))
	if builtQuery == nil {
		return []*model.Entry{}, 0, nil
	}
	results, total, err := s.repo.Search(span.Context(), builtQuery, limit, offset, utils.StringToSlice(sortBy, s.cfg.Get().Search.Defaults.SortBy))
	results = s.removeBlocked(results)
	log.Info().
		Err(err).
		Str("query", q).
		Int("limit", limit).
		Int("offset", offset).
		Int("results", len(results)).
		Int("total", total).
		Msg("search request")
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

func (s *Search) getEmptyQueryResults(ctx context.Context, limit, offset int) (entries []*model.Entry, length int) {
	rooms := s.data.GetBiggestRooms(ctx, limit, offset)
	entries = make([]*model.Entry, 0, len(rooms))
	for _, room := range rooms {
		entries = append(entries, room.Entry())
	}

	return entries, len(entries)
}

// removeBlocked removes results from blocked servers from the search results
func (s *Search) removeBlocked(results []*model.Entry) []*model.Entry {
	if len(results) == 0 {
		return results
	}

	allowed := []*model.Entry{}
	for _, entry := range results {
		if entry.IsBlocked(s.block) {
			continue
		}

		allowed = append(allowed, entry)
	}

	return allowed
}

func (s *Search) matchFields(queryStr string) (sanitizedQuery string, fields map[string]string) {
	if !strings.Contains(queryStr, ":") { // if no key:value pair(-s) - nothing is here
		return queryStr, nil
	}
	fields = map[string]string{}
	parts := strings.Split(queryStr, " ") // e.g. "language:EN foss"
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
		queryStr = strings.ReplaceAll(queryStr, remove, "")
	}
	queryStr = strings.TrimSpace(queryStr)

	return queryStr, fields
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
	stopwords := s.cfg.Get().Blocklist.Queries
	for k, v := range fields {
		if slices.Contains(stopwords, k) {
			return true
		}
		if slices.Contains(stopwords, v) {
			return true
		}
	}
	for _, k := range strings.Split(q, " ") {
		if slices.Contains(stopwords, k) {
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
