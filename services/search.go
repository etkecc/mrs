package services

import (
	"gitlab.com/etke.cc/mrs/api/model"
)

// Search service
type Search struct {
	repo SearchRepository
}

// SearchRepository interface
type SearchRepository interface {
	Search(query string, limit, offset int, sortBy []string) ([]*model.Entry, error)
}

// NewSearch creates new search service
func NewSearch(repo SearchRepository) Search {
	return Search{repo}
}

// Search things
// ref: https://blevesearch.com/docs/Query-String-Query/
func (s Search) Search(query string, limit, offset int, sortBy []string) ([]*model.Entry, error) {
	return s.repo.Search(query, limit, offset, sortBy)
}
