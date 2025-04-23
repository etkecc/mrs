package matrix

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

var mediaFallbacks = []string{"https://matrix-client.matrix.org"}

// GetClientWellKnown returns json-eligible response for /.well-known/matrix/client
func (s *Server) GetClientWellKnown() []byte {
	return s.wellknownClient
}

// GetSupportWellKnown returns json-eligible response for /.well-known/matrix/support
func (s *Server) GetSupportWellKnown() []byte {
	return s.wellknownSupport
}

// GetClientVersion returns json-eligible response for /_matrix/client/versions
func (s *Server) GetClientVersion() []byte {
	return s.versionClient
}

// GetClientDirectory is /_matrix/client/v3/directory/room/{roomAlias}
func (s *Server) GetClientDirectory(ctx context.Context, alias string) (statusCode int, respb []byte) {
	span := utils.StartSpan(ctx, "matrix.GetClientDirectory")
	defer span.Finish()
	log := zerolog.Ctx(span.Context())
	alias = utils.Unescape(alias)

	log.Info().Str("alias", alias).Str("origin", "client").Msg("querying directory")
	if alias == "" {
		return http.StatusBadRequest, s.getErrorResp(span.Context(), "M_INVALID_PARAM", "Room alias invalid")
	}

	var room *model.MatrixRoom
	s.data.EachRoom(span.Context(), func(_ string, data *model.MatrixRoom) bool {
		if data.Alias == alias {
			room = data
			return true
		}
		return false
	})
	if room == nil {
		return http.StatusNotFound, s.getErrorResp(span.Context(), "M_NOT_FOUND", "room not found")
	}

	resp := &queryDirectoryResp{
		RoomID:  room.ID,
		Servers: room.Servers(s.cfg.Get().Matrix.ServerName),
	}
	respb, err := utils.JSON(resp)
	if err != nil {
		log.Error().Err(err).Msg("cannot marshal query directory resp")
		return http.StatusInternalServerError, nil
	}

	return http.StatusOK, respb
}

// GetClientRoomSummary is /_matrix/client/unstable/is.nheko.summary/summary/{roomIdOrAlias}
func (s *Server) GetClientRoomSummary(ctx context.Context, aliasOrID string) (statusCode int, resp []byte) {
	span := utils.StartSpan(ctx, "matrix.GetClientRoomSummary")
	defer span.Finish()
	log := zerolog.Ctx(span.Context())
	aliasOrID = utils.Unescape(aliasOrID)

	log.Info().Str("aliasOrID", aliasOrID).Str("origin", "client").Msg("getting room summary")
	if aliasOrID == "" {
		return http.StatusBadRequest, s.getErrorResp(span.Context(), "M_INVALID_PARAM", "Room alias or id is invalid")
	}

	var room *model.MatrixRoom
	s.data.EachRoom(span.Context(), func(_ string, data *model.MatrixRoom) bool {
		if data.Alias == aliasOrID || data.ID == aliasOrID {
			room = data
			return true
		}
		return false
	})
	if room == nil {
		return http.StatusNotFound, s.getErrorResp(span.Context(), "M_NOT_FOUND", "room not found")
	}
	respb, err := utils.JSON(room.DirectoryEntry())
	if err != nil {
		log.Error().Err(err).Msg("cannot marshal room into room directory entry")
		return http.StatusInternalServerError, nil
	}

	return http.StatusOK, respb
}

// GetClientRoomVisibility is /_matrix/client/v3/directory/list/room/{roomID}
func (s *Server) GetClientRoomVisibility(ctx context.Context, id string) (statusCode int, resp []byte) {
	span := utils.StartSpan(ctx, "matrix.GetClientRoomVisibility")
	defer span.Finish()
	log := zerolog.Ctx(span.Context())
	id = utils.Unescape(id)

	room, err := s.data.GetRoom(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("room", id).Msg("cannot get room")
		return http.StatusInternalServerError, s.getErrorResp(span.Context(), "M_INTERNAL_ERROR", "internal error")
	}
	if room == nil {
		return http.StatusNotFound, s.getErrorResp(span.Context(), "M_NOT_FOUND", "room not found")
	}

	resp, err = utils.JSON(map[string]string{"visibility": "public"}) // MRS doesn't have any other
	if err != nil {
		log.Error().Err(err).Str("room", id).Msg("cannot marshal room visibility")
		return http.StatusInternalServerError, s.getErrorResp(span.Context(), "M_INTERNAL_ERROR", "internal error")
	}
	return http.StatusOK, resp
}
