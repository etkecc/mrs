package services

import (
	"context"
	"net/http"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/etkecc/go-apm"
	"github.com/etkecc/go-kit"
	"golang.org/x/exp/slices"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/model/mcontext"
)

// Search service
type Search struct {
	cfg       ConfigService
	data      searchDataRepository
	repo      SearchRepository
	stats     StatsService
	block     BlocklistService
	plausible PlausibleService
	stopwords map[string]bool
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
func NewSearch(cfg ConfigService, data searchDataRepository, repo SearchRepository, block BlocklistService, stats StatsService, plausible PlausibleService) *Search {
	s := &Search{
		cfg:       cfg,
		data:      data,
		repo:      repo,
		stats:     stats,
		block:     block,
		plausible: plausible,
	}
	s.initStopwords()

	return s
}

// Search things
// ref: https://blevesearch.com/docs/Query-String-Query/
func (s *Search) Search(ctx context.Context, req *http.Request, q, sortBy string, roomTypes []string, limit, offset int) ([]*model.Entry, int, error) {
	log := apm.Log(ctx)
	originServer := mcontext.GetOrigin(ctx)
	highlights := s.availableHighlights(originServer)
	if limit == 0 {
		limit = s.cfg.Get().Search.Defaults.Limit
	}
	if offset == 0 {
		offset = s.cfg.Get().Search.Defaults.Offset
	}
	limit -= highlights
	if limit == 0 {
		limit = 1
	}
	offset -= highlights
	if offset < 0 {
		offset = 0
	}

	var builtQuery query.Query
	if q == "" {
		entries, length := s.getEmptyQueryResults(ctx, roomTypes, limit, offset)
		entries = s.addHighlights(originServer, entries)
		return entries, length, nil
	}
	q, fields, fuzzy := s.matchFields(q)
	q = strings.TrimPrefix(strings.TrimSpace(q), "#")
	qTrack := strings.TrimSpace(strings.ToLower(q))
	if qTrack != "" {
		evt := model.NewAnalyticsEvent(ctx, "Search", map[string]string{"query": qTrack}, req)
		go func(ctx context.Context, evt *model.AnalyticsEvent) {
			ctx = context.WithoutCancel(ctx)
			s.plausible.Track(ctx, evt)
		}(ctx, evt)
	}

	builtQuery = s.getSearchQuery(q, fields, roomTypes, fuzzy)
	if builtQuery == nil {
		return []*model.Entry{}, 0, nil
	}
	results, total, err := s.repo.Search(ctx, builtQuery, limit, offset, kit.StringToSlice(sortBy, s.cfg.Get().Search.Defaults.SortBy))
	results = s.addHighlights(originServer, s.removeBlocked(results))
	log.Info().
		Err(err).
		Str("query", q).
		Int("limit", limit).
		Int("offset", offset).
		Int("results", len(results)).
		Int("total", total).
		Any("query", builtQuery).
		Msg("search request")
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

func (s *Search) availableHighlights(originServer string) int {
	if originServer == "" {
		return 0
	}
	highlights := s.cfg.Get().Search.Highlights
	if len(highlights) == 0 {
		return 0
	}
	count := 0
	for _, highlight := range highlights {
		if slices.Contains(highlight.Servers, originServer) {
			count++
		}
	}

	return count
}

func (s *Search) addHighlights(originServer string, entries []*model.Entry) []*model.Entry {
	if len(entries) == 0 {
		return entries
	}

	highlights := s.cfg.Get().Search.Highlights
	for _, highlight := range highlights {
		if !slices.Contains(highlight.Servers, originServer) {
			continue
		}

		entry := highlight.Entry()
		if highlight.Position < 0 || highlight.Position > len(entries) {
			entries = append(entries, entry)
			continue
		}
		entries = append(entries[:highlight.Position], append([]*model.Entry{entry}, entries[highlight.Position:]...)...)
	}

	return entries
}

func (s *Search) getEmptyQueryResults(ctx context.Context, roomTypes []string, limit, offset int) (entries []*model.Entry, length int) {
	rooms := s.data.GetBiggestRooms(ctx, limit, offset)
	entries = make([]*model.Entry, 0, len(rooms))

	// If no filter provided, return both rooms and spaces as-is.
	if len(roomTypes) == 0 {
		for _, room := range rooms {
			entries = append(entries, room.Entry())
		}
		return entries, len(entries)
	}

	includeRegular := false
	allowed := make(map[string]struct{}, len(roomTypes))
	for _, rt := range roomTypes {
		if rt == "" {
			includeRegular = true
			continue
		}
		allowed[rt] = struct{}{}
	}

	for _, room := range rooms {
		// regular rooms have empty RoomType
		if room.RoomType == "" && includeRegular {
			entries = append(entries, room.Entry())
			continue
		}
		if _, ok := allowed[room.RoomType]; ok {
			entries = append(entries, room.Entry())
		}
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

// matchFields parses the query string and returns the sanitized query string, fields and fuzzy flag
// fuzzy flag is a field "fuzzy" in the fields map, if it is set to "true" (default), then the query should be treated as fuzzy
func (s *Search) matchFields(queryStr string) (sanitizedQuery string, fields map[string]string, fuzzy bool) {
	if !strings.Contains(queryStr, ":") { // if no key:value pair(-s) - nothing is here
		return queryStr, nil, true
	}
	fields = map[string]string{"fuzzy": "true"}
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
		fields[strings.TrimSpace(strings.ToLower(pair[0]))] = strings.TrimSpace(pair[1])
	}

	for _, remove := range toRemove {
		queryStr = strings.ReplaceAll(queryStr, remove, "")
	}
	queryStr = strings.TrimSpace(queryStr)
	isFuzzy := strings.EqualFold(fields["fuzzy"], "true")
	delete(fields, "fuzzy") // it's not a real field, but a flag, so remove it to not confuse bleve

	return queryStr, fields, isFuzzy
}

func (s *Search) getSearchQuery(q string, fields map[string]string, roomTypes []string, fuzzy bool) query.Query {
	// base/standard query
	if s.shouldReject(q, fields) {
		return nil
	}
	if q == "" {
		return s.getFieldsQuery(fields)
	}

	phrase := strings.Contains(q, " ")
	queries := []query.Query{
		s.newMatchQuery(q, "name", phrase),
		s.newMatchQuery(q, "alias", phrase),
		s.newMatchQuery(q, "topic", phrase),
		s.newMatchQuery(q, "server", phrase),
	}

	if fuzzy {
		queries = append(queries,
			s.newFuzzyQuery(q, "name"),
			s.newFuzzyQuery(q, "alias"),
			s.newFuzzyQuery(q, "topic"),
			s.newFuzzyQuery(q, "server"),
		)
	}

	var mainQ query.Query
	mainQ = bleve.NewDisjunctionQuery(queries...)
	// optional room types
	roomTypeQ := s.newRoomTypeQuery(roomTypes)
	if roomTypeQ != nil {
		mainQ = bleve.NewConjunctionQuery(mainQ, roomTypeQ)
	}
	// optional fields, like "language:EN"
	fieldsQ := s.getFieldsQuery(fields)
	if fieldsQ == nil {
		return mainQ
	}
	return bleve.NewConjunctionQuery(mainQ, fieldsQ)
}

func (s *Search) getFieldsQuery(fields map[string]string) query.Query {
	if len(fields) == 0 {
		return nil
	}
	boolQ := bleve.NewBooleanQuery()
	for field, fieldQ := range fields {
		boolQ.AddMust(s.newTermQuery(fieldQ, field))
	}
	return boolQ
}

// initStopwords initializes the stopwords map from the configuration
func (s *Search) initStopwords() {
	s.stopwords = map[string]bool{}
	for _, word := range s.cfg.Get().Blocklist.Queries {
		word = strings.ToLower(strings.TrimSpace(word))
		if word != "" {
			s.stopwords[word] = true
		}
	}
}

// shouldReject checks if query or fields contain words from the stoplist
func (s *Search) shouldReject(q string, fields map[string]string) bool {
	for k, v := range fields {
		v = strings.ToLower(strings.TrimSpace(v))
		if s.stopwords[k] || s.stopwords[v] {
			return true
		}
	}
	for _, k := range strings.Split(strings.ToLower(q), " ") {
		k = strings.TrimSpace(k)
		if s.stopwords[k] {
			return true
		}
	}
	return false
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

// newRoomTypeQuery creates a query that behaves like room_type IN(roomTypes),
// with a special case: "" means "regular rooms" (documents that are NOT m.space).
func (s *Search) newRoomTypeQuery(roomTypes []string) query.Query {
	if len(roomTypes) == 0 {
		return nil
	}

	includeRegular := false
	typed := make([]query.Query, 0, len(roomTypes))

	for _, rt := range roomTypes {
		if rt == "" {
			includeRegular = true
			continue
		}
		// keep using the helper to set field and (optional) boost policy
		typed = append(typed, s.newTermQuery(rt, "room_type"))
	}

	if includeRegular {
		// regular rooms = NOT room_type:"m.space"
		notSpace := bleve.NewBooleanQuery()
		notSpace.AddMust(bleve.NewMatchAllQuery())
		sp := bleve.NewTermQuery("m.space")
		sp.SetField("room_type")
		notSpace.AddMustNot(sp)

		if len(typed) == 0 {
			// Only "" requested -> just regular rooms
			return notSpace
		}
		// "" plus explicit types -> union of (NOT m.space) OR (room_type in typed)
		return bleve.NewDisjunctionQuery(append(typed, notSpace)...)
	}

	if len(typed) == 1 {
		return typed[0]
	}
	return bleve.NewDisjunctionQuery(typed...)
}

func (s *Search) newTermQuery(match, field string) bleveQuery {
	searchQuery := bleve.NewTermQuery(match)
	searchQuery.SetField(field)
	searchQuery.SetBoost(SearchFieldsBoost[field])

	return searchQuery
}

func (s *Search) newFuzzyQuery(match, field string) bleveQuery {
	searchQuery := bleve.NewFuzzyQuery(match)
	searchQuery.SetField(field)
	searchQuery.SetBoost(SearchFieldsBoost[field])

	return searchQuery
}
