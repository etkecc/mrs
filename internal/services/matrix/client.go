package matrix

import (
	"context"
	"net/http"

	"github.com/etkecc/go-apm"
	"github.com/etkecc/mrs/internal/utils"
)

var (
	mediaFallbacks = []string{"https://matrix-client.matrix.org"}
	roomVisibility = utils.MustJSON(map[string]string{"visibility": "public"})
)

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
	log := apm.Log(ctx)
	alias = utils.Unescape(alias)

	log.Info().Str("alias", alias).Str("origin", "client").Msg("querying directory")
	if alias == "" {
		return http.StatusBadRequest, s.getErrorResp(ctx, "M_INVALID_PARAM", "Room alias invalid")
	}

	room, err := s.getRoom(ctx, alias)
	if err != nil {
		log.Error().Err(err).Msg("cannot get room from data store")
	}
	if room == nil {
		return http.StatusNotFound, s.getErrorResp(ctx, "M_NOT_FOUND", "room not found")
	}

	resp := &queryDirectoryResp{
		RoomID:  room.ID,
		Servers: room.Servers(s.cfg.Get().Matrix.ServerName),
	}
	respb, err = utils.JSON(resp)
	if err != nil {
		log.Error().Err(err).Msg("cannot marshal query directory resp")
		return http.StatusInternalServerError, nil
	}

	return http.StatusOK, respb
}

// GetClientRoomSummary is /_matrix/client/unstable/is.nheko.summary/summary/{roomIdOrAlias}
func (s *Server) GetClientRoomSummary(ctx context.Context, aliasOrID string) (statusCode int, resp []byte) {
	log := apm.Log(ctx)
	aliasOrID = utils.Unescape(aliasOrID)

	log.Info().Str("aliasOrID", aliasOrID).Str("origin", "client").Msg("getting room summary")
	if aliasOrID == "" {
		return http.StatusBadRequest, s.getErrorResp(ctx, "M_INVALID_PARAM", "Room alias or id is invalid")
	}

	room, err := s.getRoom(ctx, aliasOrID)
	if err != nil {
		log.Error().Err(err).Msg("cannot get room from data store")
	}
	if room == nil {
		return http.StatusNotFound, s.getErrorResp(ctx, "M_NOT_FOUND", "room not found")
	}
	respb, err := utils.JSON(room.DirectoryEntry())
	if err != nil {
		log.Error().Err(err).Msg("cannot marshal room into room directory entry")
		return http.StatusInternalServerError, nil
	}

	return http.StatusOK, respb
}

// GetClientRoomVisibility is /_matrix/client/v3/directory/list/room/{roomID}
// this is a stub endpoint, because MRS works only with public rooms,
// so we always return public visibility.
// That may change in the future (e.g., if protocol adds more visibility types),
// but for now we just return public visibility.
func (s *Server) GetClientRoomVisibility(_ context.Context, _ string) (statusCode int, resp []byte) {
	return http.StatusOK, roomVisibility
}
