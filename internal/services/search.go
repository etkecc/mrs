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
	"language":    100,
	"name":        10,
	"name_exact":  20, // 2x name: bias an exact full-word hit above the stemmed hit beside it (TF-IDF still has a vote)
	"server":      10,
	"alias":       5,
	"topic":       3,
	"topic_exact": 6, // 2x topic, same reason
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
		// empty query is a directory listing (biggest rooms); track it as a Search too, or federation
		// publicRooms browsing without a filter stays invisible, which is most of the directory traffic.
		s.trackSearch(ctx, req, "")
		entries, length := s.getEmptyQueryResults(ctx, roomTypes, limit, offset)
		entries = s.addHighlights(originServer, entries)
		return entries, length, nil
	}
	q, fields, fuzzy := s.matchFields(q)
	q = strings.TrimPrefix(strings.TrimSpace(q), "#")
	qTrack := strings.TrimSpace(strings.ToLower(q))
	if qTrack != "" {
		s.trackSearch(ctx, req, qTrack)
	}

	builtQuery = s.getSearchQuery(q, fields, roomTypes, fuzzy)
	if builtQuery == nil {
		return []*model.Entry{}, 0, nil
	}
	sort := kit.StringToSlice(sortBy, s.cfg.Get().Search.Defaults.SortBy)
	if q == "" {
		// Filter-only queries (e.g. "language:EN") have equal _score for all
		// matches. Append -members as a deterministic tiebreaker so pagination
		// stays stable.
		sort = append(sort, "-members")
	}
	results, total, err := s.repo.Search(ctx, builtQuery, limit, offset, sort)
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

// trackSearch fires a fire-and-forget Search analytics event; WithoutCancel so the request finishing doesn't kill the send.
func (s *Search) trackSearch(ctx context.Context, req *http.Request, term string) {
	evt := model.NewAnalyticsEvent(ctx, "Search", map[string]string{"query": term}, req)
	go func(ctx context.Context, evt *model.AnalyticsEvent) {
		ctx = context.WithoutCancel(ctx)
		s.plausible.Track(ctx, evt)
	}(ctx, evt)
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
	total := s.stats.Get().Rooms.Indexed

	if len(roomTypes) == 0 {
		return s.biggestRoomsEntries(ctx, limit, offset), total
	}

	includeRegular, allowed := parseRoomTypeFilter(roomTypes)
	return s.filteredBiggestRoomsEntries(ctx, includeRegular, allowed, limit, offset), total
}

func (s *Search) roomTypeMatches(roomType string, includeRegular bool, allowed map[string]struct{}) bool {
	if roomType == "" {
		return includeRegular
	}
	_, ok := allowed[roomType]
	return ok
}

// biggestRoomsEntries converts the paginated biggest-rooms slice into search entries.
func (s *Search) biggestRoomsEntries(ctx context.Context, limit, offset int) []*model.Entry {
	rooms := s.data.GetBiggestRooms(ctx, limit, offset)
	entries := make([]*model.Entry, 0, len(rooms))
	for _, room := range rooms {
		entries = append(entries, room.Entry())
	}
	return entries
}

// parseRoomTypeFilter splits the requested room types into regular-room and typed-room filters.
func parseRoomTypeFilter(roomTypes []string) (includeRegular bool, allowed map[string]struct{}) {
	allowed = make(map[string]struct{}, len(roomTypes))
	for _, rt := range roomTypes {
		if rt == "" {
			includeRegular = true
			continue
		}
		allowed[rt] = struct{}{}
	}
	return includeRegular, allowed
}

// filteredBiggestRoomsEntries paginates the biggest-rooms list and keeps only rooms matching the room type filter.
func (s *Search) filteredBiggestRoomsEntries(
	ctx context.Context,
	includeRegular bool,
	allowed map[string]struct{},
	limit, offset int,
) []*model.Entry {
	entries := make([]*model.Entry, 0, limit)
	skipped := 0
	for fetchOffset := 0; len(entries) < limit; fetchOffset += emptyQueryBatchSize(limit) {
		rooms := s.data.GetBiggestRooms(ctx, emptyQueryBatchSize(limit), fetchOffset)
		if len(rooms) == 0 {
			break
		}
		entries, skipped = s.appendFilteredRoomEntries(entries, rooms, includeRegular, allowed, limit, offset, skipped)
	}
	return entries
}

// emptyQueryBatchSize expands filtered empty-query scans to reduce repeated data fetches.
func emptyQueryBatchSize(limit int) int {
	batchSize := limit * 5
	if batchSize < 50 {
		return 50
	}
	return batchSize
}

// appendFilteredRoomEntries applies room-type filtering and offset skipping to a fetched room batch.
func (s *Search) appendFilteredRoomEntries(
	entries []*model.Entry,
	rooms []*model.MatrixRoom,
	includeRegular bool,
	allowed map[string]struct{},
	limit, offset, skipped int,
) (updatedEntries []*model.Entry, updatedSkipped int) {
	updatedEntries = entries
	updatedSkipped = skipped
	for _, room := range rooms {
		if !s.roomTypeMatches(room.RoomType, includeRegular, allowed) {
			continue
		}
		if updatedSkipped < offset {
			updatedSkipped++
			continue
		}
		updatedEntries = append(updatedEntries, room.Entry())
		if len(updatedEntries) >= limit {
			break
		}
	}
	return updatedEntries, updatedSkipped
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
	if s.shouldReject(q, fields) {
		return nil
	}
	if q == "" {
		return combineQueries(s.getFieldsQuery(fields), s.newRoomTypeQuery(roomTypes))
	}

	mainQuery := query.Query(bleve.NewDisjunctionQuery(s.buildTextSearchQueries(q, fuzzy)...))
	mainQuery = combineQueries(mainQuery, s.newRoomTypeQuery(roomTypes))
	return combineQueries(mainQuery, s.getFieldsQuery(fields))
}

// combineQueries joins optional query fragments without wrapping nil operands.
func combineQueries(left, right query.Query) query.Query {
	switch {
	case left == nil:
		return right
	case right == nil:
		return left
	default:
		return bleve.NewConjunctionQuery(left, right)
	}
}

// buildTextSearchQueries assembles match, prefix, and fuzzy clauses for free-text search.
func (s *Search) buildTextSearchQueries(q string, fuzzy bool) []query.Query {
	phrase := strings.Contains(q, " ")
	words := strings.Fields(q)
	searchFields := []string{"name", "alias", "topic", "server"}
	queries := make([]query.Query, 0, len(searchFields)*(2+len(words)*2))

	for _, field := range searchFields {
		queries = append(queries, s.newMatchQuery(q, field, false))
		if phrase {
			queries = append(queries, s.newMatchQuery(q, field, true))
		}
	}

	// unstemmed twins of name/topic: match-only, so a full inflected query still
	// lands when the stemmer clipped the indexed word. No prefix/fuzzy here, those
	// already ride the stemmed fields and fuzzy across 4 fields burns CPU for nothing.
	for _, field := range []string{"name_exact", "topic_exact"} {
		queries = append(queries, s.newMatchQuery(q, field, false))
	}

	queries = s.appendPrefixQueries(queries, words, searchFields)
	if fuzzy {
		queries = s.appendFuzzyQueries(queries, words, searchFields)
	}

	return queries
}

// appendPrefixQueries adds prefix clauses for each token/field combination.
func (s *Search) appendPrefixQueries(queries []query.Query, words, searchFields []string) []query.Query {
	for _, word := range words {
		for _, field := range searchFields {
			queries = append(queries, s.newPrefixQuery(word, field))
		}
	}
	return queries
}

// appendFuzzyQueries adds fuzzy clauses for each token/field combination.
func (s *Search) appendFuzzyQueries(queries []query.Query, words, searchFields []string) []query.Query {
	for _, word := range words {
		for _, field := range searchFields {
			queries = append(queries, s.newFuzzyQuery(word, field))
		}
	}
	return queries
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
	boost := SearchFieldsBoost[field]
	if phrase {
		boost *= 2 // reward exact phrase ordering
	}
	searchQuery.SetBoost(boost)

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

func (s *Search) newPrefixQuery(match, field string) bleveQuery {
	searchQuery := bleve.NewPrefixQuery(strings.ToLower(match))
	searchQuery.SetField(field)
	searchQuery.SetBoost(SearchFieldsBoost[field])

	return searchQuery
}
