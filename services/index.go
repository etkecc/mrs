package services

import (
	"log"
	"sync"

	"github.com/blevesearch/bleve/v2"

	"gitlab.com/etke.cc/mrs/api/model"
)

type Index struct {
	sync.Mutex

	index IndexRepository
	data  DataRepository
	batch *bleve.Batch
}

type IndexRepository interface {
	Index(roomID string, data *model.Entry) error
	IndexBatch(*bleve.Batch) error
	NewBatch() *bleve.Batch
}

// NewIndex creates new index service
func NewIndex(index IndexRepository, data DataRepository) *Index {
	batch := index.NewBatch()
	return &Index{
		index: index,
		data:  data,
		batch: batch,
	}
}

// RoomsBatch indexes rooms in batches
func (i *Index) RoomsBatch(size int, roomID string, data *model.Entry) error {
	i.Lock()
	defer i.Unlock()

	if i.batch.Size() >= size {
		return i.IndexBatch()
	}

	return i.batch.Index(roomID, data)
}

// IndexBatch performs indexing of the current batch
func (i *Index) IndexBatch() error {
	size := i.batch.Size()
	log.Println("indexing batch of", size, "ops")
	err := i.index.IndexBatch(i.batch)
	i.batch.Reset()
	log.Println("indexed batch of", size, "ops")
	return err
}
