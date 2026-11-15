package kit

import (
	"cmp"
	"sync"
)

// List is a concurrency-safe, order-forgetting set returned sorted; zero value unusable, build via NewList/NewListFrom.
type List[T cmp.Ordered, V any] struct {
	mu   *sync.Mutex
	data map[T]struct{}
}

// NewList creates an empty List; both type parameters must be explicit. Use NewListFrom to start from a slice.
func NewList[T cmp.Ordered, V any]() *List[T, V] {
	return &List[T, V]{
		mu:   &sync.Mutex{},
		data: make(map[T]struct{}),
	}
}

// NewListFrom creates a List from slice, owning a deduplicated copy; V is fixed to T since no map is involved.
func NewListFrom[T cmp.Ordered](slice []T) *List[T, T] {
	list := NewList[T, T]()
	list.AddSlice(slice)
	return list
}

// AddMapKeys adds all keys from datamap to the List; duplicates are ignored.
func (l *List[T, V]) AddMapKeys(datamap map[T]V) {
	for k := range datamap {
		l.Add(k)
	}
}

// AddSlice adds all elements from dataslice to the List; duplicates are ignored.
func (l *List[T, V]) AddSlice(dataslice []T) {
	for _, k := range dataslice {
		l.Add(k)
	}
}

// Add inserts item into the List; a no-op if it already exists.
func (l *List[T, V]) Add(item T) {
	l.mu.Lock()
	if _, ok := l.data[item]; !ok {
		l.data[item] = struct{}{}
	}
	l.mu.Unlock()
}

// RemoveSlice removes all elements of dataslice from the List; missing elements are ignored.
func (l *List[T, V]) RemoveSlice(dataslice []T) {
	for _, k := range dataslice {
		l.Remove(k)
	}
}

// Remove deletes item from the List; a no-op if it doesn't exist.
func (l *List[T, V]) Remove(item T) {
	l.mu.Lock()
	delete(l.data, item)
	l.mu.Unlock()
}

// Len returns the item count.
func (l *List[T, V]) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.data)
}

// Slice returns the List's contents as a fresh, ascending-sorted slice, via MapKeys.
func (l *List[T, V]) Slice() []T {
	l.mu.Lock()
	defer l.mu.Unlock()
	return MapKeys(l.data)
}
