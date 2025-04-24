package services

import (
	"strings"
	"sync"

	"github.com/etkecc/go-kit"
	"golang.org/x/exp/slices"
)

// Blocklist service
type Blocklist struct {
	mu      *sync.Mutex
	cfg     ConfigService
	dynamic map[string]struct{}
}

// NewBlocklist creates new blocklist service
func NewBlocklist(cfg ConfigService) *Blocklist {
	return &Blocklist{
		mu:      &sync.Mutex{},
		cfg:     cfg,
		dynamic: map[string]struct{}{},
	}
}

// Len of the blocklist
func (b *Blocklist) Len() int {
	return len(b.cfg.Get().Blocklist.Servers) + len(b.dynamic)
}

// Slice returns slice of the static+dynamic blocklist
func (b *Blocklist) Slice() []string {
	return kit.Uniq(append(kit.MapKeys(b.dynamic), b.cfg.Get().Blocklist.Servers...))
}

// Reset dynamic part of the blocklist
func (b *Blocklist) Reset() {
	b.mu.Lock()
	b.dynamic = map[string]struct{}{}
	b.mu.Unlock()
}

// Add server to blocklist
func (b *Blocklist) Add(server string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.dynamic[server] = struct{}{}
}

// ByID checks if server of matrixID is present in the blocklist
func (b *Blocklist) ByID(matrixID string) bool {
	idx := strings.LastIndex(matrixID, ":")
	if idx == -1 {
		return false
	}
	if idx+2 == len(matrixID) { // "wrongid:"
		return false
	}
	server := matrixID[idx+1:]

	return b.ByServer(server)
}

// ByServer checks if server is present in the blocklist
func (b *Blocklist) ByServer(server string) bool {
	if slices.Contains(b.cfg.Get().Blocklist.Servers, server) {
		return true
	}
	if _, ok := b.dynamic[server]; ok {
		return true
	}
	return false
}
