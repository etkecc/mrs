package matrix

import (
	"net/http"
	"strconv"
	"time"

	"gitlab.com/etke.cc/mrs/api/metrics"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

// GetServerWellKnown returns json-eligible response for /.well-known/matrix/server
func (s *Server) GetServerWellKnown() []byte {
	return s.wellknownServer
}

// GetServerVersion returns json-eligible response for /_matrix/federation/v1/version
func (s *Server) GetServerVersion() []byte {
	return s.versionServer
}

// GetKeyServer returns jsonblob-eligible response for /_matrix/key/v2/server
func (s *Server) GetKeyServer() []byte {
	resp := s.keyServer
	resp.ValidUntilTS = time.Now().UTC().Add(24 * 7 * time.Hour).UnixMilli()
	payload, err := s.signJSON(resp)
	if err != nil {
		utils.Logger.Error().Err(err).Msg("cannot sign payload")
	}
	return payload
}

// PublicRooms returns /_matrix/federation/v1/publicRooms response
func (s *Server) PublicRooms(req *http.Request, rdReq *model.RoomDirectoryRequest) (int, []byte) {
	origin, err := s.ValidateAuth(req)
	if err != nil {
		utils.Logger.Warn().Err(err).Msg("matrix auth failed")
		return http.StatusUnauthorized, nil
	}

	defer metrics.IncSearchQueries("matrix", origin)

	limit := rdReq.Limit
	if limit == 0 {
		limit = MatrixSearchLimit
	}
	if limit > MatrixSearchLimit {
		limit = s.cfg.Get().Search.Defaults.Limit
	}
	offset := utils.StringToInt(rdReq.Since)
	entries, total, err := s.search.Search(rdReq.Filter.GenericSearchTerm, "", limit, offset)
	if err != nil {
		utils.Logger.Error().Err(err).Msg("search from matrix failed")
		return http.StatusInternalServerError, nil
	}
	chunk := make([]*model.RoomDirectoryRoom, 0, len(entries))
	for _, entry := range entries {
		chunk = append(chunk, entry.RoomDirectory())
	}

	var prev int
	if offset >= limit {
		prev = offset - limit
	}
	var next int
	if len(chunk) >= limit {
		next = offset + len(chunk)
	}

	var prevBatch string
	if prev > 0 {
		prevBatch = strconv.Itoa(prev)
	}

	var nextBatch string
	if next > 0 {
		nextBatch = strconv.Itoa(next)
	}

	value, err := utils.JSON(model.RoomDirectoryResponse{
		Chunk:     chunk,
		PrevBatch: prevBatch,
		NextBatch: nextBatch,
		Total:     total,
	})
	if err != nil {
		utils.Logger.Error().Err(err).Msg("cannot marshal room directory json")
		return http.StatusInternalServerError, nil
	}
	return http.StatusOK, value
}
