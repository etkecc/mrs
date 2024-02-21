package batch

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"gitlab.com/etke.cc/mrs/api/utils"
)

// Batch struct
type Batch[T any] struct {
	sync.Mutex
	flushfunc func(ctx context.Context, items []T)
	data      []T
	size      int
}

// New creates new batch object
func New[T any](size int, flushfunc func(ctx context.Context, items []T)) *Batch[T] {
	return &Batch[T]{
		data:      make([]T, 0, size),
		flushfunc: flushfunc,
		size:      size,
	}
}

// Add items from channel to batch and automatically flush them
func (b *Batch[T]) Add(ctx context.Context, item T) {
	b.Lock()
	b.data = append(b.data, item)
	b.Unlock()

	if len(b.data) >= b.size {
		b.Flush(ctx)
	}
}

// Flush / store batch
func (b *Batch[T]) Flush(ctx context.Context) {
	b.Lock()
	defer b.Unlock()

	span := utils.StartSpan(ctx, "batch.Flush")
	defer span.Finish()
	log := zerolog.Ctx(span.Context())

	started := time.Now().UTC()
	log.Info().Int("len", len(b.data)).Msg("storing data batch")
	b.flushfunc(span.Context(), b.data)
	log.Info().Int("len", len(b.data)).Str("took", time.Since(started).String()).Msg("stored data batch")
	b.data = make([]T, 0, b.size)
}
