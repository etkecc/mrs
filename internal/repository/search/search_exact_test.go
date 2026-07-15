package search

import (
	"context"
	"os"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/pemistahl/lingua-go"

	"github.com/etkecc/mrs/internal/model"
)

// multilangRooms are faked from live rooms: identifiers placeholder, name/topic
// verbatim so the stemmers trip like prod. Kept off testEntries, count asserts depend on it.
var multilangRooms = []*model.Entry{
	{ // the actual reported bug: German prose, place names the de stemmer clips -en from
		ID: "!de-lugvs:example.org", Type: "room", Alias: "#de-lugvs:example.org",
		Name:   "LUG Villingen-Schwenningen",
		Topic:  "Der Raum der Linux User Group Villingen-Schwenningen für alle Themen rund um Linux und freie Software in der Region.",
		Server: "example.org", Members: 100, Language: "DE", JoinRule: "public",
	},
	{
		ID: "!de-info:example.org", Type: "room", Alias: "#de-info:example.org",
		Name: "TU Graz - Info", Topic: "In diesem Raum werden Informationen der TU Graz veröffentlicht.",
		Server: "example.org", Members: 100, Language: "DE", JoinRule: "public",
	},
	{
		ID: "!ru-intl:example.org", Type: "room", Alias: "#ru-intl:example.org",
		Name: "Международный чат русскоязычных пользователей Matrix", Topic: "Свобода слова.",
		Server: "example.org", Members: 100, Language: "RU", JoinRule: "public",
	},
	{
		ID: "!ru-lang:example.org", Type: "room", Alias: "#ru-lang:example.org",
		Name: "Russian", Topic: "дружественное сообщество, ориентированное на изучение русского языка и обмен знаниями",
		Server: "example.org", Members: 100, Language: "RU", JoinRule: "public",
	},
	{
		ID: "!fr-suse:example.org", Type: "room", Alias: "#fr-suse:example.org",
		Name: "openSUSE Français", Topic: "Communauté francophone de openSUSE",
		Server: "example.org", Members: 100, Language: "FR", JoinRule: "public",
	},
	{
		ID: "!fr-lamyne:example.org", Type: "room", Alias: "#fr-lamyne:example.org",
		Name: "accueil-et-presentation", Topic: "Canal daccueil des nouveaux et nouvelles venu.e.s destiné aux présentations des un.e.s",
		Server: "example.org", Members: 100, Language: "FR", JoinRule: "public",
	},
	{
		ID: "!es-suse:example.org", Type: "room", Alias: "#es-suse:example.org",
		Name: "openSUSE Español", Topic: "Comunidad habla hispana de openSUSE",
		Server: "example.org", Members: 100, Language: "ES", JoinRule: "public",
	},
	{
		ID: "!en-plain:example.org", Type: "room", Alias: "#en-plain:example.org",
		Name: "Postmoogle Bridge", Topic: "an email bridge for Matrix homeservers",
		Server: "example.org", Members: 100, Language: "EN", JoinRule: "public",
	},
	{ // "The Matrix": proves "the" never rides the exact field as a boost-20 token
		ID: "!en-the:example.org", Type: "room", Alias: "#en-the:example.org",
		Name: "The Matrix", Topic: "the film discussion room for the classic",
		Server: "example.org", Members: 100, Language: "EN", JoinRule: "public",
	},
}

// testLangDetector is the one detector the whole test binary shares. multilang.Register
// registers into a global registry, first-writer-wins, so a second detector would be
// silently ignored: every test builder must hand NewIndex this exact set.
func testLangDetector() lingua.LanguageDetector {
	return lingua.NewLanguageDetectorBuilder().
		FromLanguages(lingua.English, lingua.German, lingua.French, lingua.Russian, lingua.Spanish).
		Build()
}

// newIndexWith builds a fresh index over the given entries, sharing the global detector.
func newIndexWith(t *testing.T, entries []*model.Entry) *Index {
	t.Helper()

	dir, err := os.MkdirTemp("", "mrs-search-exact-test-*")
	if err != nil {
		t.Fatal("failed to create temp dir:", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	idx, err := NewIndex(dir+"/index", testLangDetector(), "en")
	if err != nil {
		t.Fatal("failed to create index:", err)
	}
	t.Cleanup(func() { idx.Close() })

	batch := idx.NewBatch()
	for _, entry := range entries {
		if err := batch.Index(entry.ID, entry); err != nil {
			t.Fatal("failed to add to batch:", err)
		}
	}
	if err := idx.IndexBatch(batch); err != nil {
		t.Fatal("failed to index batch:", err)
	}

	return idx
}

func searchOneField(t *testing.T, idx *Index, q, field string) []*model.Entry {
	t.Helper()
	mq := bleve.NewMatchQuery(q)
	mq.SetField(field)
	results, _, err := idx.Search(context.Background(), mq, 20, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	return results
}

// TestSearch_ExactMatchAllLanguages: the full inflected word matches its unstemmed
// *_exact field in every language. This is the fix's behavior, language-general.
func TestSearch_ExactMatchAllLanguages(t *testing.T) {
	idx := newIndexWith(t, multilangRooms)

	tests := []struct{ name, query, field, wantID string }{
		{"de villingen", "villingen", "topic_exact", "!de-lugvs:example.org"},
		{"de schwenningen", "schwenningen", "topic_exact", "!de-lugvs:example.org"},
		{"de informationen", "Informationen", "topic_exact", "!de-info:example.org"},
		{"ru пользователей (name)", "пользователей", "name_exact", "!ru-intl:example.org"},
		{"ru знаниями", "знаниями", "topic_exact", "!ru-lang:example.org"},
		{"fr communauté", "Communauté", "topic_exact", "!fr-suse:example.org"},
		{"fr présentations", "présentations", "topic_exact", "!fr-lamyne:example.org"},
		{"es comunidad", "Comunidad", "topic_exact", "!es-suse:example.org"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits := searchOneField(t, idx, tt.query, tt.field)
			if !containsID(hits, tt.wantID) {
				t.Fatalf("exact field %q for %q: want %q, got %v", tt.field, tt.query, tt.wantID, searchIDs(hits))
			}
		})
	}
}

// TestSearch_ExactBugWitness: the SAME full word misses the stemmed base field, so
// the exact field is load-bearing, not redundant. Proven on the reported German case,
// where the de stemmer clips -en from the place names but an English query can't.
func TestSearch_ExactBugWitness(t *testing.T) {
	idx := newIndexWith(t, multilangRooms)

	tests := []struct{ name, query, stemField, wantID string }{
		{"villingen", "villingen", "topic", "!de-lugvs:example.org"},
		{"informationen", "Informationen", "topic", "!de-info:example.org"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits := searchOneField(t, idx, tt.query, tt.stemField)
			if containsID(hits, tt.wantID) {
				t.Fatalf("bug-witness broken: %q matched stemmed %q, so the exact field isn't load-bearing here", tt.query, tt.stemField)
			}
		})
	}
}

// TestSearch_ExactTruncationsStillWork: a truncation into the stem still prefix-matches
// the stemmed field, the recall path the fix must not regress.
func TestSearch_ExactTruncationsStillWork(t *testing.T) {
	idx := newIndexWith(t, multilangRooms)

	pq := bleve.NewPrefixQuery("villing")
	pq.SetField("topic")
	results, _, err := idx.Search(context.Background(), pq, 20, 0, []string{"_score"})
	if err != nil {
		t.Fatal("Search error:", err)
	}
	if !containsID(results, "!de-lugvs:example.org") {
		t.Fatalf("truncation %q should still prefix-match stemmed topic, got %v", "villing", searchIDs(results))
	}
}

// TestSearch_ExactEnglishUnaffected: English name/topic still match plainly.
func TestSearch_ExactEnglishUnaffected(t *testing.T) {
	idx := newIndexWith(t, multilangRooms)

	if hits := searchOneField(t, idx, "Postmoogle", "name"); !containsID(hits, "!en-plain:example.org") {
		t.Fatalf("english name should match, got %v", searchIDs(hits))
	}
	if hits := searchOneField(t, idx, "email", "topic"); !containsID(hits, "!en-plain:example.org") {
		t.Fatalf("english topic should match, got %v", searchIDs(hits))
	}
}

// TestSearch_ExactStopwordSanity: "the" is stopword-filtered off the exact field,
// the meaningful token still lands.
func TestSearch_ExactStopwordSanity(t *testing.T) {
	idx := newIndexWith(t, multilangRooms)

	if hits := searchOneField(t, idx, "the", "name_exact"); containsID(hits, "!en-the:example.org") {
		t.Fatalf(`"the" must not match name_exact of "The Matrix", got %v`, searchIDs(hits))
	}
	if hits := searchOneField(t, idx, "Matrix", "name_exact"); !containsID(hits, "!en-the:example.org") {
		t.Fatalf(`"Matrix" should match name_exact of "The Matrix", got %v`, searchIDs(hits))
	}
}
