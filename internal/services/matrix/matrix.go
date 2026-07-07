package matrix

import (
	"context"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/etkecc/mrs/internal/model"
)

const (
	MatrixSearchLimit = 100 // default matrix (!) search limit
	devhost           = "localhost"
	cacheTTL          = 24 * time.Hour // matches the Matrix .well-known/matrix/server default TTL
	// namesNegativeCacheTTL is short on purpose: long enough to swallow a repeat-probe flood, short enough a
	// host that blinked offline for a minute isn't written off for the whole day.
	namesNegativeCacheTTL = 5 * time.Minute
)

// Server server
type Server struct {
	cfg                configService
	keys               []*model.Key
	data               dataRepository
	media              mediaService
	search             searchService
	blocklist          blocklistService
	wellknownServer    []byte        // /.well-known/matrix/server contents
	wellknownClient    []byte        // /.well-known/matrix/client contents
	wellknownSupport   []byte        // /.well-known/matrix/support contents
	versionServer      []byte        // /_matrix/federation/v1/version contents
	versionClient      []byte        // /_matrix/client/versions contents
	keyServer          matrixKeyResp // /_matrix/key/v2/server template
	discoverFunc       func(context.Context, string) int
	discoverSem        chan struct{}
	surlsCache         *expirable.LRU[string, string]
	curlsCache         *expirable.LRU[string, string]
	keysCache          *expirable.LRU[string, matrixKeyResp]
	namesCache         *expirable.LRU[string, string]
	namesNegativeCache *expirable.LRU[string, struct{}] // recently-failed name lookups, short-TTL'd
}

// NewServer creates new matrix server
func NewServer(cfg configService, data dataRepository, media mediaService, search searchService, blocklist blocklistService) (*Server, error) {
	keysCache := expirable.NewLRU[string, matrixKeyResp](100000, nil, cacheTTL)
	namesCache := expirable.NewLRU[string, string](100000, nil, cacheTTL)
	surlsCache := expirable.NewLRU[string, string](100000, nil, cacheTTL)
	curlsCache := expirable.NewLRU[string, string](100000, nil, cacheTTL)
	namesNegativeCache := expirable.NewLRU[string, struct{}](100000, nil, namesNegativeCacheTTL)

	s := &Server{
		cfg:                cfg,
		data:               data,
		media:              media,
		search:             search,
		blocklist:          blocklist,
		surlsCache:         surlsCache,
		curlsCache:         curlsCache,
		keysCache:          keysCache,
		namesCache:         namesCache,
		namesNegativeCache: namesNegativeCache,
	}
	// bound concurrent fire-and-forget discovery spawns to the configured worker count (floor 1)
	s.discoverSem = make(chan struct{}, max(1, cfg.Get().Workers.Discovery))
	if err := s.initWellKnown(cfg.Get().Public.API); err != nil {
		return nil, err
	}
	if err := s.initKeys(cfg.Get().Matrix.Keys); err != nil {
		return nil, err
	}
	if err := s.initVersion(); err != nil {
		return nil, err
	}
	s.initKeyServer()

	return s, nil
}

// SetDiscover func
func (s *Server) SetDiscover(discover func(context.Context, string) int) {
	s.discoverFunc = discover
}
