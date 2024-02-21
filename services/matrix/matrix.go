package matrix

import (
	"context"
	"crypto/ed25519"

	lru "github.com/hashicorp/golang-lru/v2"

	"gitlab.com/etke.cc/mrs/api/model"
)

const (
	MatrixSearchLimit = 100 // default matrix (!) search limit
	devhost           = "localhost"
)

// Server server
type Server struct {
	cfg             configService
	keys            []*model.Key
	data            dataRepository
	search          searchService
	wellknownServer []byte        // /.well-known/matrix/server contents
	wellknownClient []byte        // /.well-known/matrix/client contents
	versionServer   []byte        // /_matrix/federation/v1/version contents
	versionClient   []byte        // /_matrix/client/versions contents
	keyServer       matrixKeyResp // /_matrix/key/v2/server template
	discoverFunc    func(context.Context, string) int
	surlsCache      *lru.Cache[string, string]
	curlsCache      *lru.Cache[string, string]
	keysCache       *lru.Cache[string, map[string]ed25519.PublicKey]
	namesCache      *lru.Cache[string, string]
}

// NewServer creates new matrix server
func NewServer(cfg configService, data dataRepository, search searchService) (*Server, error) {
	keysCache, err := lru.New[string, map[string]ed25519.PublicKey](100000)
	if err != nil {
		return nil, err
	}
	namesCache, err := lru.New[string, string](100000)
	if err != nil {
		return nil, err
	}
	surlsCache, err := lru.New[string, string](100000)
	if err != nil {
		return nil, err
	}
	curlsCache, err := lru.New[string, string](100000)
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:        cfg,
		data:       data,
		search:     search,
		surlsCache: surlsCache,
		curlsCache: curlsCache,
		keysCache:  keysCache,
		namesCache: namesCache,
	}
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
