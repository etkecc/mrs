package services

import (
	"strings"
	"sync"

	"gitlab.com/etke.cc/mrs/api/utils"
)

// Blocklist service
type Blocklist struct {
	mu      *sync.Mutex
	static  map[string]struct{}
	dynamic map[string]struct{}
}

// NewBlocklist creates new blocklist service
func NewBlocklist(static []string) *Blocklist {
	static = utils.Uniq(static)
	servers := make(map[string]struct{}, len(static))
	for _, server := range static {
		servers[server] = struct{}{}
	}
	return &Blocklist{
		mu:      &sync.Mutex{},
		static:  servers,
		dynamic: map[string]struct{}{},
	}
}

// Len of the blocklist
func (b *Blocklist) Len() int {
	return len(b.static) + len(b.dynamic)
}

// Slice returns slice of the static+dynamic blocklist
func (b *Blocklist) Slice() []string {
	return utils.Uniq(append(utils.MapKeys(b.dynamic), utils.MapKeys(b.static)...))
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
	if _, ok := b.static[server]; ok {
		return true
	}
	if _, ok := b.dynamic[server]; ok {
		return true
	}
	return false
}
