package services

import (
	"github.com/blevesearch/bleve/v2"

	"gitlab.com/etke.cc/mrs/api/model"
)

// Search service
type Search struct {
	repo SearchRepository
}

// SearchRepository interface
type SearchRepository interface {
	Search(query string, limit, offset int) ([]*model.Entry, error)
	Index(roomID string, data *model.Entry) error
	IndexBatch(*bleve.Batch) error
	NewBatch() *bleve.Batch
}

// NewSearch creates new search service
func NewSearch(repo SearchRepository) Search {
	return Search{repo}
}

// Search things
// ref: https://blevesearch.com/docs/Query-String-Query/
func (s Search) Search(query string, limit, offset int) ([]*model.Entry, error) {
	return s.repo.Search(query, limit, offset)
}

// Index data
func (s Search) Index(roomID string, data *model.Entry) error {
	return s.repo.Index(roomID, data)
}

// IndexBatch of entries
func (s Search) IndexBatch(batch *bleve.Batch) error {
	return s.repo.IndexBatch(batch)
}

// NewBatch creates new batch
func (s Search) NewBatch() *bleve.Batch {
	return s.repo.NewBatch()
}
