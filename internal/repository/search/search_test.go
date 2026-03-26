package search

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/pemistahl/lingua-go"

	"github.com/etkecc/mrs/internal/model"
)

// testEntries returns a representative set of entries from etke.cc rooms
var testEntries = []*model.Entry{
	{
		ID:        "!GKRrhSQkiZgqGyhwXa:etke.cc",
		Type:      "room",
		Alias:     "#service:etke.cc",
		Name:      "etke.cc",
		Topic:     "Managed Matrix servers, including Matrix hosting, Matrix bridges, Matrix bots, Fediverse components, etc.",
		Server:    "etke.cc",
		Members:   908,
		Language:  "EN",
		RoomType:  "m.space",
		JoinRule:  "public",
		AvatarURL: "https://example.com/avatar/etke.cc/VvMCbCBlIcuesBvszBUIMLxp",
	},
	{
		ID:        "!ENsoUfnVRWEfjtSjsS:etke.cc",
		Type:      "room",
		Alias:     "#admins:etke.cc",
		Name:      "Matrix Admins",
		Topic:     "This space is designated for admins of Matrix servers, and everyone who'd like to talk to them.",
		Server:    "etke.cc",
		Members:   80,
		Language:  "EN",
		RoomType:  "m.space",
		JoinRule:  "public",
		AvatarURL: "https://example.com/avatar/etke.cc/GAnGpsMGGWgJBthAIYpyWdGI",
	},
	{
		ID:       "!LpivaKUdewaGfawMoR:etke.cc",
		Type:     "room",
		Alias:    "#synapse-admin:etke.cc",
		Name:     "Synapse Admin",
		Topic:    "etke.cc's Synapse Admin fork community room. https://github.com/etkecc/synapse-admin https://admin.etke.cc",
		Server:   "etke.cc",
		Members:  182,
		Language: "EN",
		RoomType: "",
		JoinRule: "public",
	},
	{
		ID:       "!RikgWhAbCrfXLlFaNY:etke.cc",
		Type:     "room",
		Alias:    "#updates:etke.cc",
		Name:     "etke.cc | updates",
		Topic:    "Once the Update Notifier posts a message about an update, you can install it on your server by running Synapse Admin",
		Server:   "etke.cc",
		Members:  124,
		Language: "EN",
		RoomType: "",
		JoinRule: "public",
	},
	{
		ID:       "!XODRhTLplrymaFicdK:etke.cc",
		Type:     "room",
		Alias:    "#ttm:etke.cc",
		Name:     "Time-To-Matrix",
		Topic:    "https://github.com/etkecc/ttm news and discussions.",
		Server:   "etke.cc",
		Members:  60,
		Language: "EN",
		RoomType: "",
		JoinRule: "public",
	},
	{
		ID:       "!XpiGHbsblpxcKvDUnZ:etke.cc",
		Type:     "room",
		Alias:    "#source:etke.cc",
		Name:     "etke.cc | open source",
		Topic:    "open source projects of etke.cc - https://github.com/etkecc https://liberapay.com/etkecc",
		Server:   "etke.cc",
		Members:  132,
		Language: "EN",
		RoomType: "m.space",
		JoinRule: "public",
	},
	{
		ID:       "!ZVIOUxtbNRapDHyAQK:etke.cc",
		Type:     "room",
		Alias:    "#components:etke.cc",
		Name:     "components releases",
		Topic:    "RSS subscription to components we use and provide as part of etke.cc",
		Server:   "etke.cc",
		Members:  50,
		Language: "EN",
		RoomType: "",
		JoinRule: "public",
	},
	{
		ID:       "!RnVMFDEcuWtBkYjnUS:etke.cc",
		Type:     "room",
		Alias:    "#buscarron:etke.cc",
		Name:     "Buscarron",
		Topic:    "https://github.com/etkecc/buscarron news & discussions",
		Server:   "etke.cc",
		Members:  35,
		Language: "EN",
		RoomType: "",
		JoinRule: "public",
	},
	{
		ID:       "!vMHrTKpEfwWkosbMER:etke.cc",
		Type:     "room",
		Alias:    "#honoroit:etke.cc",
		Name:     "Honoroit",
		Topic:    "https://github.com/etkecc/honoroit news & discussions",
		Server:   "etke.cc",
		Members:  95,
		Language: "EN",
		RoomType: "",
		JoinRule: "public",
	},
	{
		ID:       "!fkhmtSHXjxDMsdakCR:etke.cc",
		Type:     "room",
		Alias:    "#postmoogle:etke.cc",
		Name:     "Postmoogle",
		Topic:    "https://github.com/etkecc/postmoogle discussion & news",
		Server:   "etke.cc",
		Members:  174,
		Language: "EN",
		RoomType: "",
		JoinRule: "public",
	},
	{
		ID:       "!gqlCuoCdhufltluRXk:etke.cc",
		Type:     "room",
		Alias:    "#news:etke.cc",
		Name:     "etke.cc | news",
		Topic:    "Updates about YOUR Matrix server, 1 message per week.",
		Server:   "etke.cc",
		Members:  1001,
		Language: "EN",
		RoomType: "",
		JoinRule: "public",
	},
	{
		ID:       "!qBd432nNs7_2LYhmDRWH_6XQa1mBYJG11egyJB05pGQ",
		Type:     "room",
		Alias:    "#muninn-hall-recommended-policy-lists:codestorm.net",
		Name:     "Muninn Hall Recommended Policy Lists",
		Topic:    "",
		Server:   "etke.cc",
		Members:  36,
		Language: "-",
		RoomType: "m.space",
		JoinRule: "public",
	},
	{
		ID:       "!plLpAYoLXJuOyxjYwx:etke.cc",
		Type:     "room",
		Alias:    "#baibot:etke.cc",
		Name:     "Baibot discussion",
		Topic:    "Baibot is an AI (LLM) bot supporting various capabilities (text-generation, text-to-speech, speech-to-text, image-generation) on various providers.",
		Server:   "etke.cc",
		Members:  131,
		Language: "EN",
		RoomType: "",
		JoinRule: "public",
	},
}

// newTestIndex creates a temporary Bleve index with all test entries indexed
func newTestIndex(t *testing.T) *Index {
	t.Helper()

	dir, err := os.MkdirTemp("", "mrs-search-test-*")
	if err != nil {
		t.Fatal("failed to create temp dir:", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	detector := lingua.NewLanguageDetectorBuilder().
		FromLanguages(lingua.English, lingua.German).
		Build()

	idx, err := NewIndex(dir+"/index", detector, "en")
	if err != nil {
		t.Fatal("failed to create index:", err)
	}
	t.Cleanup(func() { idx.Close() })

	batch := idx.NewBatch()
	for _, entry := range testEntries {
		if err := batch.Index(entry.ID, entry); err != nil {
			t.Fatal("failed to add to batch:", err)
		}
	}
	if err := idx.IndexBatch(batch); err != nil {
		t.Fatal("failed to index batch:", err)
	}

	return idx
}

func searchIDs(results []*model.Entry) []string {
	ids := make([]string, 0, len(results))
	for _, r := range results {
		ids = append(ids, r.ID)
	}
	return ids
}

func containsID(results []*model.Entry, id string) bool {
	for _, r := range results {
		if r.ID == id {
			return true
		}
	}
	return false
}

func containsName(results []*model.Entry, name string) bool {
	for _, r := range results {
		if r.Name == name {
			return true
		}
	}
	return false
}

func assertSingleResultMatches(t *testing.T, results []*model.Entry, total int, want *model.Entry) {
	t.Helper()

	if total != 1 {
		t.Fatalf("expected 1 result, got %d", total)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 returned entry, got %d", len(results))
	}
	if !reflect.DeepEqual(results[0], want) {
		t.Fatalf("result mismatch:\n got: %#v\nwant: %#v", results[0], want)
	}
}

func assertContainsID(t *testing.T, results []*model.Entry, id string) {
	t.Helper()

	if !containsID(results, id) {
		t.Fatalf("expected result %q in %v", id, searchIDs(results))
	}
}

func TestIndex_Len(t *testing.T) {
	idx := newTestIndex(t)
	if got := idx.Len(); got != len(testEntries) {
		t.Errorf("Len() = %d, want %d", got, len(testEntries))
	}
}

func TestIndex_IndexAndDelete(t *testing.T) {
	idx := newTestIndex(t)

	entry := &model.Entry{
		ID:       "!test:example.com",
		Type:     "room",
		Name:     "Test Room",
		Server:   "example.com",
		RoomType: "",
		JoinRule: "public",
	}
	if err := idx.Index(entry.ID, entry); err != nil {
		t.Fatal("Index() error:", err)
	}
	if got := idx.Len(); got != len(testEntries)+1 {
		t.Errorf("after Index(), Len() = %d, want %d", got, len(testEntries)+1)
	}

	if err := idx.Delete(entry.ID); err != nil {
		t.Fatal("Delete() error:", err)
	}
	if got := idx.Len(); got != len(testEntries) {
		t.Errorf("after Delete(), Len() = %d, want %d", got, len(testEntries))
	}
}

func TestSearch_ByName(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	tests := []struct {
		name       string
		query      string
		wantInName string
	}{
		{"exact name", "Postmoogle", "Postmoogle"},
		{"exact name lowercase", "postmoogle", "Postmoogle"},
		{"partial name via different words", "Synapse Admin", "Synapse Admin"},
		{"name with special chars", "Buscarron", "Buscarron"},
		{"name with pipe", "etke.cc | news", "etke.cc | news"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := bleve.NewMatchQuery(tt.query)
			q.SetField("name")
			results, total, err := idx.Search(ctx, q, 10, 0, []string{"_score"})
			if err != nil {
				t.Fatal("Search error:", err)
			}
			if total == 0 {
				t.Fatal("expected results, got 0")
			}
			if !containsName(results, tt.wantInName) {
				t.Fatalf("expected %q in results, got IDs: %v", tt.wantInName, searchIDs(results))
			}
		})
	}
}

func TestSearch_ByTopic(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	q := bleve.NewMatchQuery("Matrix hosting bridges bots")
	q.SetField("topic")
	results, total, err := idx.Search(ctx, q, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total == 0 {
		t.Fatal("expected results for topic search, got 0")
	}
	// The etke.cc space has "Matrix hosting, Matrix bridges, Matrix bots" in topic
	assertContainsID(t, results, "!GKRrhSQkiZgqGyhwXa:etke.cc")
}

func TestSearch_ByAlias(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// matrix_alias analyzer strips #, !, : and lowercases
	q := bleve.NewMatchQuery("postmoogle")
	q.SetField("alias")
	results, total, err := idx.Search(ctx, q, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total == 0 {
		t.Fatal("expected results for alias search, got 0")
	}
	assertContainsID(t, results, "!fkhmtSHXjxDMsdakCR:etke.cc")
}

func TestSearch_ByServer(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// server is a keyword field - exact match only
	q := bleve.NewTermQuery("etke.cc")
	q.SetField("server")
	results, total, err := idx.Search(ctx, q, 100, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total != len(testEntries) {
		t.Errorf("expected all %d entries for server:etke.cc, got %d", len(testEntries), total)
	}
	if len(results) != len(testEntries) {
		t.Errorf("expected %d results, got %d", len(testEntries), len(results))
	}
}

func TestSearch_ByLanguage(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	q := bleve.NewTermQuery("EN")
	q.SetField("language")
	results, _, err := idx.Search(ctx, q, 100, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}

	// All entries except Muninn Hall (language: "-") should match
	for _, r := range results {
		if r.Language != "EN" {
			t.Errorf("unexpected language %q for %s", r.Language, r.Name)
		}
	}

	// Verify the "-" language entry is not returned
	if containsID(results, "!qBd432nNs7_2LYhmDRWH_6XQa1mBYJG11egyJB05pGQ") {
		t.Error("Muninn Hall (language: -) should not match language:EN")
	}
}

func TestSearch_ByRoomType(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	t.Run("spaces only", func(t *testing.T) {
		q := bleve.NewTermQuery("m.space")
		q.SetField("room_type")
		results, total, err := idx.Search(ctx, q, 100, 0, []string{"_score"})
		if err != nil {
			t.Fatal("Search error:", err)
		}
		if total != 4 {
			t.Errorf("expected 4 spaces, got %d", total)
		}
		for _, r := range results {
			if r.RoomType != "m.space" {
				t.Errorf("expected room_type=m.space, got %q for %s", r.RoomType, r.Name)
			}
		}
	})

	t.Run("regular rooms (NOT m.space)", func(t *testing.T) {
		boolQ := bleve.NewBooleanQuery()
		boolQ.AddMust(bleve.NewMatchAllQuery())
		notSpace := bleve.NewTermQuery("m.space")
		notSpace.SetField("room_type")
		boolQ.AddMustNot(notSpace)

		results, total, err := idx.Search(ctx, boolQ, 100, 0, []string{"_score"})
		if err != nil {
			t.Fatal("Search error:", err)
		}
		if total != 9 {
			t.Errorf("expected 9 regular rooms, got %d", total)
		}
		for _, r := range results {
			if r.RoomType == "m.space" {
				t.Errorf("unexpected space in regular rooms results: %s", r.Name)
			}
		}
	})
}

func TestSearch_ByJoinRule(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// join_rule is now a keyword field
	q := bleve.NewTermQuery("public")
	q.SetField("join_rule")
	_, total, err := idx.Search(ctx, q, 100, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total != len(testEntries) {
		t.Errorf("expected all %d entries with join_rule=public, got %d", len(testEntries), total)
	}
}

func TestSearch_StoredFields(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// Search for a specific room and verify all stored fields are returned
	q := bleve.NewTermQuery("!GKRrhSQkiZgqGyhwXa:etke.cc")
	q.SetField("_id")
	results, total, err := idx.Search(ctx, q, 1, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	assertSingleResultMatches(t, results, total, testEntries[0])
}

func TestSearch_Pagination(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	q := bleve.NewMatchAllQuery()

	// Page 1
	results1, total, err := idx.Search(ctx, q, 5, 0, []string{"-members"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total != len(testEntries) {
		t.Errorf("total = %d, want %d", total, len(testEntries))
	}
	if len(results1) != 5 {
		t.Errorf("page 1 len = %d, want 5", len(results1))
	}

	// Page 2
	results2, _, err := idx.Search(ctx, q, 5, 5, []string{"-members"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if len(results2) != 5 {
		t.Errorf("page 2 len = %d, want 5", len(results2))
	}

	// No overlap between pages
	page1IDs := make(map[string]bool)
	for _, r := range results1 {
		page1IDs[r.ID] = true
	}
	for _, r := range results2 {
		if page1IDs[r.ID] {
			t.Errorf("room %s appears on both page 1 and page 2", r.ID)
		}
	}

	// First page sorted by -members should have the highest member count first
	if results1[0].Members < results1[len(results1)-1].Members {
		t.Errorf("page 1 not sorted by -members: first=%d, last=%d", results1[0].Members, results1[len(results1)-1].Members)
	}
}

func TestSearch_SortByMembers(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	q := bleve.NewMatchAllQuery()
	results, _, err := idx.Search(ctx, q, 100, 0, []string{"-members"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if len(results) != len(testEntries) {
		t.Fatalf("expected %d results, got %d", len(testEntries), len(results))
	}

	// Verify descending member order
	for i := 1; i < len(results); i++ {
		if results[i-1].Members < results[i].Members {
			t.Errorf("not sorted by -members at position %d: %d < %d (%s vs %s)",
				i, results[i-1].Members, results[i].Members, results[i-1].Name, results[i].Name)
		}
	}
}

func TestSearch_NoIndexFieldsNotSearchable(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// avatar_url is noindex - searching it should return nothing
	q := bleve.NewTermQuery("https://example.com/avatar/etke.cc/VvMCbCBlIcuesBvszBUIMLxp")
	q.SetField("avatar_url")
	_, total, err := idx.Search(ctx, q, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total != 0 {
		t.Errorf("expected 0 results for noindex field search, got %d", total)
	}
}

func TestSearch_DisjunctionQuery(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// Simulate the kind of query the search service builds:
	// search "Postmoogle" across name, alias, topic
	nameQ := bleve.NewMatchQuery("Postmoogle")
	nameQ.SetField("name")
	nameQ.SetBoost(10)

	aliasQ := bleve.NewMatchQuery("Postmoogle")
	aliasQ.SetField("alias")
	aliasQ.SetBoost(5)

	topicQ := bleve.NewMatchQuery("Postmoogle")
	topicQ.SetField("topic")
	topicQ.SetBoost(3)

	disjunction := bleve.NewDisjunctionQuery(nameQ, aliasQ, topicQ)
	results, total, err := idx.Search(ctx, disjunction, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total == 0 {
		t.Fatal("expected results for disjunction query")
	}
	// Postmoogle room should be the top result (name match with highest boost)
	if results[0].ID != "!fkhmtSHXjxDMsdakCR:etke.cc" {
		t.Errorf("expected Postmoogle as top result, got %s (%s)", results[0].Name, results[0].ID)
	}
}

func TestSearch_ConjunctionWithRoomType(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// Search "etke" but only in spaces
	nameQ := bleve.NewMatchQuery("etke")
	nameQ.SetField("name")

	roomTypeQ := bleve.NewTermQuery("m.space")
	roomTypeQ.SetField("room_type")

	conjunction := bleve.NewConjunctionQuery(nameQ, roomTypeQ)
	results, _, err := idx.Search(ctx, conjunction, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}

	for _, r := range results {
		if r.RoomType != "m.space" {
			t.Errorf("expected only spaces, got room_type=%q for %s", r.RoomType, r.Name)
		}
	}
}

func TestSearch_FuzzyQuery(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// Typo: "Postmoogl" should still find "Postmoogle"
	q := bleve.NewFuzzyQuery("postmoogl")
	q.SetField("name")
	results, total, err := idx.Search(ctx, q, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total == 0 {
		t.Fatal("expected fuzzy results for 'postmoogl'")
	}
	if !containsName(results, "Postmoogle") {
		t.Fatalf("expected Postmoogle in fuzzy results, got IDs: %v", searchIDs(results))
	}
}

func TestSearch_PrefixQuery(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// "postm" should match "Postmoogle" via prefix
	q := bleve.NewPrefixQuery("postm")
	q.SetField("name")
	results, total, err := idx.Search(ctx, q, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total == 0 {
		t.Fatal("expected prefix results for 'postm'")
	}
	if !containsName(results, "Postmoogle") {
		t.Fatalf("expected Postmoogle in prefix results, got IDs: %v", searchIDs(results))
	}
}

func TestSearch_MatchPhraseQuery(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	q := bleve.NewMatchPhraseQuery("Synapse Admin")
	q.SetField("name")
	results, total, err := idx.Search(ctx, q, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total == 0 {
		t.Fatal("expected phrase match results for 'Synapse Admin'")
	}
	if !containsName(results, "Synapse Admin") {
		t.Fatalf("expected 'Synapse Admin' in phrase match results, got IDs: %v", searchIDs(results))
	}
}

func TestSearch_EmptyQueryWithRoomTypeFilter(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// Simulate the fixed empty-query + room_types path:
	// MatchAll filtered to only spaces
	roomTypeQ := bleve.NewTermQuery("m.space")
	roomTypeQ.SetField("room_type")

	results, total, err := idx.Search(ctx, roomTypeQ, 100, 0, []string{"-members"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total != 4 {
		t.Errorf("expected 4 spaces, got %d", total)
	}
	for _, r := range results {
		if r.RoomType != "m.space" {
			t.Errorf("expected only spaces, got room_type=%q for %s", r.RoomType, r.Name)
		}
	}

	// Verify pagination works correctly with filtered results
	page1, total1, err := idx.Search(ctx, roomTypeQ, 2, 0, []string{"-members"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	page2, _, err := idx.Search(ctx, roomTypeQ, 2, 2, []string{"-members"})
	if err != nil {
		t.Fatal("Search error:", err)
	}

	if total1 != 4 {
		t.Errorf("total = %d, want 4", total1)
	}
	if len(page1) != 2 {
		t.Errorf("page1 len = %d, want 2", len(page1))
	}
	if len(page2) != 2 {
		t.Errorf("page2 len = %d, want 2", len(page2))
	}

	// No overlap
	for _, r1 := range page1 {
		for _, r2 := range page2 {
			if r1.ID == r2.ID {
				t.Errorf("room %s appears on both pages", r1.ID)
			}
		}
	}
}

func TestSearch_RegularRoomsFilter(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// Simulate room_types: [null] → regular rooms only (NOT m.space)
	boolQ := bleve.NewBooleanQuery()
	boolQ.AddMust(bleve.NewMatchAllQuery())
	notSpace := bleve.NewTermQuery("m.space")
	notSpace.SetField("room_type")
	boolQ.AddMustNot(notSpace)

	results, total, err := idx.Search(ctx, boolQ, 100, 0, []string{"-members"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total != 9 {
		t.Errorf("expected 9 regular rooms, got %d", total)
	}
	for _, r := range results {
		if r.RoomType == "m.space" {
			t.Errorf("unexpected space in regular rooms: %s", r.Name)
		}
	}
}

func TestSearch_CombinedSearchAndRoomType(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// Search "admin" in regular rooms only (not spaces)
	nameQ := bleve.NewMatchQuery("admin")
	nameQ.SetField("name")

	topicQ := bleve.NewMatchQuery("admin")
	topicQ.SetField("topic")

	searchQ := bleve.NewDisjunctionQuery(nameQ, topicQ)

	// NOT m.space
	boolQ := bleve.NewBooleanQuery()
	boolQ.AddMust(bleve.NewMatchAllQuery())
	notSpace := bleve.NewTermQuery("m.space")
	notSpace.SetField("room_type")
	boolQ.AddMustNot(notSpace)

	combined := bleve.NewConjunctionQuery(searchQ, boolQ)
	results, _, err := idx.Search(ctx, combined, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}

	for _, r := range results {
		if r.RoomType == "m.space" {
			t.Errorf("unexpected space in results: %s", r.Name)
		}
	}

	// "Synapse Admin" is a regular room and should be in results
	if !containsName(results, "Synapse Admin") {
		t.Error("expected 'Synapse Admin' in filtered results")
	}

	// "Matrix Admins" is a space and should NOT be in results
	if containsName(results, "Matrix Admins") {
		t.Error("unexpected 'Matrix Admins' (space) in regular room results")
	}
}

func TestSearch_BatchIndexing(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	batch := idx.NewBatch()
	for i := 0; i < 5; i++ {
		entry := &model.Entry{
			ID:       "!batch" + string(rune('A'+i)) + ":test.com",
			Type:     "room",
			Name:     "Batch Room " + string(rune('A'+i)),
			Server:   "test.com",
			RoomType: "",
			JoinRule: "public",
		}
		if err := batch.Index(entry.ID, entry); err != nil {
			t.Fatal("batch.Index error:", err)
		}
	}
	if err := idx.IndexBatch(batch); err != nil {
		t.Fatal("IndexBatch error:", err)
	}

	if got := idx.Len(); got != len(testEntries)+5 {
		t.Errorf("Len() = %d, want %d", got, len(testEntries)+5)
	}

	// Verify we can search the new entries
	q := bleve.NewTermQuery("test.com")
	q.SetField("server")
	_, total, err := idx.Search(ctx, q, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total != 5 {
		t.Errorf("expected 5 batch-indexed results, got %d", total)
	}
}

func TestSearch_MultiWordNonPhrase(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// "open source" as non-phrase match should find "etke.cc | open source"
	q := bleve.NewMatchQuery("open source")
	q.SetField("name")
	results, total, err := idx.Search(ctx, q, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total == 0 {
		t.Fatal("expected results for 'open source' non-phrase match")
	}
	if !containsName(results, "etke.cc | open source") {
		t.Error("expected 'etke.cc | open source' in results")
	}
}

func TestSearch_IDField(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// matrix_id analyzer strips #, !, : and lowercases
	// searching for parts of a room ID should work
	q := bleve.NewMatchQuery("etke.cc")
	q.SetField("id")
	results, total, err := idx.Search(ctx, q, 100, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total == 0 {
		t.Fatal("expected results when searching id field for 'etke.cc'")
	}
	_ = results
}

func TestSearch_ScoreBoostOrdering(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// Build a disjunction like the search service does:
	// "Baibot" should rank higher in name (boost=10) than in topic (boost=3)
	nameQ := bleve.NewMatchQuery("baibot")
	nameQ.SetField("name")
	nameQ.SetBoost(10)

	topicQ := bleve.NewMatchQuery("baibot")
	topicQ.SetField("topic")
	topicQ.SetBoost(3)

	aliasQ := bleve.NewMatchQuery("baibot")
	aliasQ.SetField("alias")
	aliasQ.SetBoost(5)

	disjunction := bleve.NewDisjunctionQuery(nameQ, topicQ, aliasQ)
	results, total, err := idx.Search(ctx, disjunction, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total == 0 {
		t.Fatal("expected results for 'baibot'")
	}
	// "Baibot discussion" should be first since it matches on name (highest boost)
	if results[0].ID != "!plLpAYoLXJuOyxjYwx:etke.cc" {
		t.Errorf("expected Baibot discussion as top result, got %s (%s)", results[0].Name, results[0].ID)
	}
}

// TestSearch_RoomTypeDisjunction tests the OR logic for room_types: [null, "m.space"]
func TestSearch_RoomTypeDisjunction(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// room_types: [null, "m.space"] → both regular rooms AND spaces
	spaceQ := bleve.NewTermQuery("m.space")
	spaceQ.SetField("room_type")

	notSpace := bleve.NewBooleanQuery()
	notSpace.AddMust(bleve.NewMatchAllQuery())
	ns := bleve.NewTermQuery("m.space")
	ns.SetField("room_type")
	notSpace.AddMustNot(ns)

	disjunction := bleve.NewDisjunctionQuery(spaceQ, notSpace)
	_, total, err := idx.Search(ctx, disjunction, 100, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	// Should return all entries
	if total != len(testEntries) {
		t.Errorf("expected all %d entries, got %d", len(testEntries), total)
	}
}

func TestSearch_FieldsQuery(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// Simulate language:EN field filter (term query on keyword field)
	q := bleve.NewTermQuery("EN")
	q.SetField("language")

	// Combined with room type filter
	roomTypeQ := bleve.NewTermQuery("m.space")
	roomTypeQ.SetField("room_type")

	combined := bleve.NewConjunctionQuery(q, roomTypeQ)
	results, _, err := idx.Search(ctx, combined, 100, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}

	for _, r := range results {
		if r.Language != "EN" {
			t.Errorf("expected language=EN, got %q for %s", r.Language, r.Name)
		}
		if r.RoomType != "m.space" {
			t.Errorf("expected room_type=m.space, got %q for %s", r.RoomType, r.Name)
		}
	}

	// Muninn Hall is a space but language="-", so it shouldn't be here
	if containsID(results, "!qBd432nNs7_2LYhmDRWH_6XQa1mBYJG11egyJB05pGQ") {
		t.Error("Muninn Hall should not match language:EN filter")
	}
}

// TestSearch_TopicMatchContributes verifies that topic matches contribute to search results
// (regression test for zero-boost bug)
func TestSearch_TopicMatchContributes(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// "AI" appears only in Baibot's topic: "Baibot is an AI (LLM) bot..."
	q := bleve.NewMatchQuery("AI")
	q.SetField("topic")
	q.SetBoost(3) // the new topic boost
	results, total, err := idx.Search(ctx, q, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total == 0 {
		t.Fatal("expected results for topic search 'AI' - topic boost may be zero")
	}

	// Only combined: "AI" should contribute to the disjunction score
	nameQ := bleve.NewMatchQuery("AI")
	nameQ.SetField("name")
	nameQ.SetBoost(10)

	topicQ := bleve.NewMatchQuery("AI")
	topicQ.SetField("topic")
	topicQ.SetBoost(3)

	disjunction := bleve.NewDisjunctionQuery(nameQ, topicQ)
	results2, total2, err := idx.Search(ctx, disjunction, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total2 == 0 {
		t.Fatal("expected results for disjunction query with topic boost")
	}
	assertContainsID(t, results2, "!plLpAYoLXJuOyxjYwx:etke.cc")
	_ = results
}

// Verify that the _all field matches are not used (we use explicit field queries)
func TestSearch_ExplicitFieldsRequired(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// A query without SetField would search _all; verify our field-specific approach works
	q := bleve.NewMatchQuery("Postmoogle")
	q.SetField("name")
	_, total, err := idx.Search(ctx, q, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total == 0 {
		t.Fatal("expected results with explicit field query")
	}

	// Same term on wrong field should return nothing
	q2 := bleve.NewMatchQuery("Postmoogle")
	q2.SetField("server")
	_, total2, err := idx.Search(ctx, q2, 10, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if total2 != 0 {
		t.Errorf("expected 0 results for 'Postmoogle' on server field, got %d", total2)
	}
}

// TestSearch_NumericMembers verifies numeric field querying works
func TestSearch_NumericMembers(t *testing.T) {
	idx := newTestIndex(t)
	ctx := context.Background()

	// Find rooms with at least 500 members
	minMembers := float64(500)
	q := query.NewNumericRangeQuery(&minMembers, nil)
	q.SetField("members")
	results, total, err := idx.Search(ctx, q, 100, 0, []string{"-members"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	// etke.cc space (908) and news (1001) have 500+ members
	if total != 2 {
		t.Errorf("expected 2 rooms with 500+ members, got %d", total)
	}
	for _, r := range results {
		if r.Members < 500 {
			t.Errorf("unexpected room with %d members: %s", r.Members, r.Name)
		}
	}
}
