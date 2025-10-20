package matrix

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/etkecc/go-apm"
	"github.com/etkecc/go-kit"

	"github.com/etkecc/mrs/internal/metrics"
	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/model/mcontext"
	"github.com/etkecc/mrs/internal/utils"
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
func (s *Server) GetKeyServer(ctx context.Context) []byte {
	log := apm.Log(ctx)

	resp := s.keyServer
	resp.ValidUntilTS = time.Now().UTC().Add(24 * 7 * time.Hour).UnixMilli()
	payload, err := s.signJSON(resp)
	if err != nil {
		log.Error().Err(err).Msg("cannot sign payload")
	}
	return payload
}

// PublicRooms returns /_matrix/federation/v1/publicRooms response
func (s *Server) PublicRooms(ctx context.Context, req *http.Request, rdReq *model.RoomDirectoryRequest) (statusCode int, resp []byte) {
	log := apm.Log(ctx)
	origin, err := s.ValidateAuth(ctx, req)
	if err != nil {
		log.Warn().Err(err).Str("header", req.Header.Get("Authorization")).Msg("matrix auth failed")
		return http.StatusUnauthorized, nil
	}
	ctx = mcontext.WithOrigin(ctx, origin)
	req.Header.Set("Referer", "https://"+origin+"/_matrix/client/v3/publicRooms") // workaround to set correct referer for this endpoint
	defer metrics.IncSearchQueries("matrix", origin)

	limit := rdReq.Limit
	if limit == 0 {
		limit = MatrixSearchLimit
	}
	if limit > MatrixSearchLimit {
		limit = s.cfg.Get().Search.Defaults.Limit
	}
	offset := kit.StringToInt(rdReq.Since)
	entries, total, err := s.search.Search(ctx, req, rdReq.Filter.GenericSearchTerm, "", rdReq.Filter.RoomTypes, limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("search from matrix failed")
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
		log.Error().Err(err).Msg("cannot marshal room directory json")
		return http.StatusInternalServerError, nil
	}
	return http.StatusOK, value
}
