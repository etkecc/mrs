package services

import (
	"log"
	"sync"

	"github.com/blevesearch/bleve/v2"

	"gitlab.com/etke.cc/mrs/api/metrics"
	"gitlab.com/etke.cc/mrs/api/model"
)

type Index struct {
	sync.Mutex

	index     IndexRepository
	data      DataRepository
	batch     *bleve.Batch
	batchSize int
}

type IndexRepository interface {
	Index(roomID string, data *model.Entry) error
	Delete(roomID string) error
	IndexBatch(*bleve.Batch) error
	NewBatch() *bleve.Batch
}

// NewIndex creates new index service
func NewIndex(index IndexRepository, data DataRepository, roomsBatchSize int) *Index {
	batch := index.NewBatch()
	return &Index{
		index:     index,
		data:      data,
		batch:     batch,
		batchSize: roomsBatchSize,
	}
}

// RoomsBatch indexes rooms in batches
func (i *Index) RoomsBatch(roomID string, data *model.Entry) error {
	i.Lock()
	defer i.Unlock()

	if i.batch.Size() >= i.batchSize {
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
	metrics.RoomsIndexed.Add(float64(size))
	return err
}
