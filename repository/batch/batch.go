package batch

import "log"

// Batch struct
type Batch[T any] struct {
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
	b.data = append(b.data, item)
	if len(b.data) >= b.size {
		b.Flush()
	}
}

// Flush / store batch
func (b *Batch[T]) Flush() {
	log.Println("data.Batch", "storing batch of", len(b.data), "items")
	b.flushfunc(b.data)
	b.data = make([]T, 0, b.size)
}
