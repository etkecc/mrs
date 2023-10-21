package services

import (
	"strings"
	"sync"

	"gitlab.com/etke.cc/mrs/api/utils"
)

// Blocklist service
type Blocklist struct {
	mu      *sync.Mutex
	servers map[string]struct{}
}

// NewBlocklist creates new blocklist service
func NewBlocklist(list []string) *Blocklist {
	list = utils.Uniq(list)
	servers := make(map[string]struct{}, len(list))
	for _, server := range list {
		servers[server] = struct{}{}
	}
	return &Blocklist{
		mu:      &sync.Mutex{},
		servers: servers,
	}
}

// Len of the blocklist
func (b *Blocklist) Len() int {
	return len(b.servers)
}

// Add server to blocklist
func (b *Blocklist) Add(server string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.servers[server] = struct{}{}
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
	if _, ok := b.servers[server]; ok {
		return true
	}
	return false
}
