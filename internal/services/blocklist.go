package services

import (
	"regexp"
	"strings"
	"sync"

	"github.com/etkecc/go-apm"
)

// Blocklist service
type Blocklist struct {
	mu      *sync.Mutex
	cfg     ConfigService
	regexes []*regexp.Regexp
	dynamic map[string]struct{}
}

// NewBlocklist creates new blocklist service
func NewBlocklist(cfg ConfigService) *Blocklist {
	b := &Blocklist{
		mu:      &sync.Mutex{},
		cfg:     cfg,
		dynamic: map[string]struct{}{},
	}
	b.initRegexes()
	return b
}

// initRegexes initializes regexes from the config
func (b *Blocklist) initRegexes() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.regexes = make([]*regexp.Regexp, 0, len(b.cfg.Get().Blocklist.Servers))
	for _, r := range b.cfg.Get().Blocklist.Servers {
		re, err := regexp.Compile(r)
		if err != nil {
			apm.Log().Error().Err(err).Str("regex", r).Msg("Failed to compile blocklist.servers regex for blocklist")
		}
		b.regexes = append(b.regexes, re)
	}
}

// Len of the blocklist
func (b *Blocklist) Len() int {
	return len(b.cfg.Get().Blocklist.Servers) + len(b.dynamic)
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
	for _, re := range b.regexes {
		if re.MatchString(server) {
			return true
		}
	}

	if _, ok := b.dynamic[server]; ok {
		return true
	}
	return false
}
