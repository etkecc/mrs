package utils

import (
	"sync"
)

// List is unique list
type List[T comparable, V any] struct {
	mu   *sync.Mutex
	data map[T]struct{}
}

// NewList creates new list
func NewList[T comparable, V any]() *List[T, V] {
	return &List[T, V]{
		mu:   &sync.Mutex{},
		data: make(map[T]struct{}),
	}
}

// AddMapKeys adds keys from map to the list
func (l *List[T, V]) AddMapKeys(datamap map[T]V) {
	for k := range datamap {
		l.Add(k)
	}
}

// AddSlice adds keys from slice to the list
func (l *List[T, V]) AddSlice(dataslice []T) {
	for _, k := range dataslice {
		l.Add(k)
	}
}

// Add item to the list
func (l *List[T, V]) Add(item T) {
	l.mu.Lock()
	if _, ok := l.data[item]; !ok {
		l.data[item] = struct{}{}
	}
	l.mu.Unlock()
}

// RemoveSlice removes items from data using slice
func (l *List[T, V]) RemoveSlice(dataslice []T) {
	for _, k := range dataslice {
		l.Remove(k)
	}
}

// Remove item from list
func (l *List[T, V]) Remove(item T) {
	l.mu.Lock()
	delete(l.data, item)
	l.mu.Unlock()
}

// Len - data length
func (l *List[T, V]) Len() int {
	return len(l.data)
}

// Slice returns list data as slice
func (l *List[T, V]) Slice() []T {
	return MapKeys(l.data)
}
