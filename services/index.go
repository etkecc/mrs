package services

import (
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

type Index struct {
	sync.Mutex

	cfg   ConfigService
	index IndexRepository
	batch *bleve.Batch
}

type IndexRepository interface {
	Index(roomID string, data *model.Entry) error
	Delete(roomID string) error
	Swap() error
	IndexBatch(*bleve.Batch) error
	NewBatch() *bleve.Batch
}

// NewIndex creates new index service
func NewIndex(cfg ConfigService, index IndexRepository) *Index {
	batch := index.NewBatch()
	return &Index{
		cfg:   cfg,
		index: index,
		batch: batch,
	}
}

// EmptyIndex creates new empty index
func (i *Index) EmptyIndex() error {
	return i.index.Swap()
}

// RoomsBatch indexes rooms in batches
func (i *Index) RoomsBatch(roomID string, data *model.Entry) error {
	i.Lock()
	defer i.Unlock()

	if i.batch.Size() >= i.cfg.Get().Batch.Rooms {
		return i.IndexBatch()
	}

	return i.batch.Index(roomID, data)
}

// IndexBatch performs indexing of the current batch
func (i *Index) IndexBatch() error {
	size := i.batch.Size()
	started := time.Now()
	utils.Logger.Info().Int("len", size).Msg("indexing batch...")
	err := i.index.IndexBatch(i.batch)
	i.batch.Reset()
	utils.Logger.Info().Int("len", size).Str("took", time.Since(started).String()).Msg("indexed batch")
	return err
}
