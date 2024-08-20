package services

import (
	"context"
	"sync"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/rs/zerolog"

	"github.com/etkecc/mrs/internal/model"
)

type Index struct {
	mu    sync.Mutex
	cfg   ConfigService
	index IndexRepository
	batch *bleve.Batch
}

type IndexRepository interface {
	Index(roomID string, data *model.Entry) error
	Delete(roomID string) error
	Swap(ctx context.Context) error
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
func (i *Index) EmptyIndex(ctx context.Context) error {
	return i.index.Swap(ctx)
}

// RoomsBatch indexes rooms in batches
func (i *Index) RoomsBatch(ctx context.Context, roomID string, data *model.Entry) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.batch.Size() >= i.cfg.Get().Batch.Rooms {
		return i.IndexBatch(ctx)
	}

	return i.batch.Index(roomID, data)
}

// IndexBatch performs indexing of the current batch
func (i *Index) IndexBatch(ctx context.Context) error {
	log := zerolog.Ctx(ctx)
	size := i.batch.Size()
	started := time.Now()
	log.Info().Int("len", size).Msg("indexing batch...")
	err := i.index.IndexBatch(i.batch)
	i.batch.Reset()
	log.Info().Int("len", size).Str("took", time.Since(started).String()).Msg("indexed batch")
	return err
}
