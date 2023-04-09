package search

import (
	"github.com/blevesearch/bleve/v2"
	"gitlab.com/etke.cc/int/mrs/model"
)

// Index new data
func (i *Index) Index(roomID string, data model.Entry) error {
	return i.index.Index(roomID, data)
}

// IndexBatch of entries
func (i *Index) IndexBatch(batch *bleve.Batch) error {
	return i.index.Batch(batch)
}

// NewBatch creates new batch
func (i *Index) NewBatch() *bleve.Batch {
	return i.index.NewBatch()
}
