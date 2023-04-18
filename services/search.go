package services

import (
	"strings"

	"github.com/pemistahl/lingua-go"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

// Search service
type Search struct {
	repo     SearchRepository
	detector lingua.LanguageDetector
}

// SearchRepository interface
type SearchRepository interface {
	Search(query string, limit, offset int) ([]*model.Entry, error)
}

// NewSearch creates new search service
func NewSearch(repo SearchRepository, detector lingua.LanguageDetector) Search {
	return Search{repo, detector}
}

// Search things
// ref: https://blevesearch.com/docs/Query-String-Query/
func (s Search) Search(query string, limit, offset int) ([]*model.Entry, error) {
	lang, confidence := utils.DetectLanguage(s.detector, query)
	// workaround for non-english queries - if query language is not english and it's determined with high confidence,
	// add * symbol (if not provided in the original query)
	if lang != "en" && confidence > 0.8 && !strings.HasSuffix(query, "*") {
		query += "*"
	}
	return s.repo.Search(query, limit, offset)
}
