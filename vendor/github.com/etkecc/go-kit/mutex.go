package kit

import (
	"sync"
)

// keyMutex pairs a per-key sync.Mutex with a reference count, tracked under the global Mutex.mu lock.
type keyMutex struct {
	mu       sync.Mutex
	refcount int
}

// Mutex is a key-based mutex, keys lock independently, entries refcounted; zero value unusable, build via NewMutex.
type Mutex struct {
	// mu protects the locks map and each keyMutex's refcount; plain Mutex since Lock both reads and writes refcount.
	mu sync.Mutex

	// locks maps each key to its per-key mutex and reference count; created on first Lock, deleted at zero.
	locks map[string]*keyMutex
}

// NewMutex creates and returns a new, initialized Mutex ready for use.
func NewMutex() *Mutex {
	return &Mutex{
		locks: make(map[string]*keyMutex),
	}
}

// Lock blocks until the per-key lock is acquired; refcount increments first so Unlock can't delete the entry mid-wait.
func (km *Mutex) Lock(key string) {
	km.mu.Lock()
	m, exists := km.locks[key]
	if !exists {
		m = &keyMutex{}
		km.locks[key] = m
	}
	m.refcount++
	km.mu.Unlock()

	m.mu.Lock()
}

// Unlock releases the per-key lock; a no-op on an unknown key, panics if the caller doesn't hold it, like sync.Mutex.
func (km *Mutex) Unlock(key string) {
	km.mu.Lock()
	m, exists := km.locks[key]
	if !exists {
		km.mu.Unlock()
		return
	}
	m.refcount--
	if m.refcount == 0 {
		delete(km.locks, key)
	}
	km.mu.Unlock()

	m.mu.Unlock()
}
