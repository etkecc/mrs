package services

import (
	"encoding/json"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/matrix-org/gomatrixserverlib"

	"gitlab.com/etke.cc/mrs/api/config"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
	"gitlab.com/etke.cc/mrs/api/version"
)

// unsignedKeyResp is unsigned response of /_matrix/key/v2/server
type unsignedKeyResp struct {
	ServerName    string                       `json:"server_name"`
	ValidUntilTS  int64                        `json:"valid_until_ts"`
	VerifyKeys    map[string]map[string]string `json:"verify_keys"`
	OldVerifyKeys map[string]any               `json:"old_verify_keys"`
}

type matrixSearchService interface {
	Search(query string, limit, offset int, sortBy []string) ([]*model.Entry, error)
}

// Matrix server
type Matrix struct {
	name      string
	keys      []*model.Key
	search    matrixSearchService
	wellknown map[string]string            // /.well-known/matrix/server contents
	version   map[string]map[string]string // /_matrix/federation/v1/version contents
	keyServer unsignedKeyResp              // /_matrix/key/v2/server template
}

// NewMatrix creates new matrix server
func NewMatrix(cfg *config.Config, search matrixSearchService) (*Matrix, error) {
	m := &Matrix{name: cfg.Matrix.ServerName, search: search}
	if err := m.initWellKnown(cfg.Public.API); err != nil {
		return nil, err
	}
	if err := m.initKeys(cfg.Matrix.Keys); err != nil {
		return nil, err
	}
	m.initVersion()
	m.initKeyServer()

	return m, nil
}

// GetWellKnown returns json-eligible response for /.well-known/matrix/server
func (m *Matrix) GetWellKnown() any {
	return m.wellknown
}

// GetVersion returns json-eligible response for /_matrix/federation/v1/version
func (m *Matrix) GetVersion() any {
	return m.version
}

// GetKeyServer returns jsonblob-eligible response for /_matrix/key/v2/server
func (m *Matrix) GetKeyServer() []byte {
	resp := m.keyServer
	resp.ValidUntilTS = time.Now().UTC().Add(24 * time.Hour).UnixMilli()
	payload, err := m.signJSON(m.keyServer)
	if err != nil {
		log.Println("ERROR: cannot sign payload:", err)
	}
	return payload
}

// PublicRooms returns /_matrix/federation/v1/publicRooms response
func (m *Matrix) PublicRooms(req *model.RoomDirectoryRequest) *model.RoomDirectoryResponse {
	limit := req.Limit
	if limit == 0 {
		limit = 30
	}
	offset := utils.StringToInt(req.Since)
	query := req.Filter.GenericSearchTerm
	if query == "" {
		query = "Matrix Rooms Search"
	}
	entries, err := m.search.Search(query, limit, offset, []string{"-members", "-_score"})
	if err != nil {
		log.Println("ERROR: cannot search from matrix:", err)
		return nil
	}
	chunk := make([]*model.RoomDirectoryRoom, 0, len(entries))
	for _, entry := range entries {
		chunk = append(chunk, entry.RoomDirectory())
	}
	var prev int
	if offset >= limit {
		prev = offset - limit
	}
	return &model.RoomDirectoryResponse{
		Chunk:     chunk,
		PrevBatch: strconv.Itoa(prev),
		NextBatch: strconv.Itoa(offset + len(chunk)),
		Total:     99999, // TODO
	}
}

func (m *Matrix) initKeys(strs []string) error {
	if len(strs) == 0 {
		return nil
	}
	keys := []*model.Key{}
	for _, str := range strs {
		key, err := model.KeyFrom(str)
		if err != nil {
			return err
		}
		keys = append(keys, key)
	}
	m.keys = keys
	return nil
}

func (m *Matrix) initWellKnown(apiURL string) error {
	uri, err := url.Parse(apiURL)
	if err != nil {
		return err
	}
	port := uri.Port()
	if port == "" {
		port = "443"
	}

	m.wellknown = map[string]string{"m.server": uri.Hostname() + ":" + port}
	return nil
}

func (m *Matrix) initVersion() {
	m.version = map[string]map[string]string{
		"server": {
			"name":    version.Name,
			"version": version.Version,
		},
	}
}

func (m *Matrix) initKeyServer() {
	resp := unsignedKeyResp{
		ServerName:    m.name,
		ValidUntilTS:  time.Now().UTC().Add(24 * time.Hour).UnixMilli(),
		VerifyKeys:    map[string]map[string]string{},
		OldVerifyKeys: map[string]any{},
	}
	for _, key := range m.keys {
		resp.VerifyKeys[key.ID] = map[string]string{"key": key.Public}
	}
	m.keyServer = resp
}

// signJSON using server keys
func (m *Matrix) signJSON(input any) ([]byte, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	for _, key := range m.keys {
		payload, err = gomatrixserverlib.SignJSON(m.name, gomatrixserverlib.KeyID(key.ID), key.Private, payload)
		if err != nil {
			return nil, err
		}
	}
	return payload, nil
}
