package batch

import (
	"sync"
	"time"

	"gitlab.com/etke.cc/mrs/api/utils"
)

// Batch struct
type Batch[T any] struct {
	sync.Mutex
	flushfunc func(items []T)
	data      []T
	size      int
}

// New creates new batch object
func New[T any](size int, flushfunc func(items []T)) *Batch[T] {
	return &Batch[T]{
		data:      make([]T, 0, size),
		flushfunc: flushfunc,
		size:      size,
	}
}

// Add items from channel to batch and automatically flush them
func (b *Batch[T]) Add(item T) {
	b.Lock()
	b.data = append(b.data, item)
	b.Unlock()

	if len(b.data) >= b.size {
		b.Flush()
	}
}

// Flush / store batch
func (b *Batch[T]) Flush() {
	b.Lock()
	started := time.Now().UTC()
	utils.Logger.Info().Int("len", len(b.data)).Msg("storing data batch")
	b.flushfunc(b.data)
	utils.Logger.Info().Int("len", len(b.data)).Str("took", time.Since(started).String()).Msg("stored data batch")
	b.data = make([]T, 0, b.size)
	b.Unlock()
}
