package matrix

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/rs/zerolog"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

var (
	mediaFallbacks         = []string{"https://matrix-client.matrix.org"}
	defaultThumbnailParams = url.Values{
		"width":        []string{"40"},
		"height":       []string{"40"},
		"method":       []string{"crop"},
		"allow_remote": []string{"true"},
	}.Encode()
)

// GetClientWellKnown returns json-eligible response for /.well-known/matrix/client
func (s *Server) GetClientWellKnown() []byte {
	return s.wellknownClient
}

// GetClientVersion returns json-eligible response for /_matrix/client/versions
func (s *Server) GetClientVersion() []byte {
	return s.versionClient
}

// GetClientDirectory is /_matrix/client/v3/directory/room/{roomAlias}
func (s *Server) GetClientDirectory(ctx context.Context, alias string) (int, []byte) {
	span := utils.StartSpan(ctx, "matrix.GetClientDirectory")
	defer span.Finish()
	log := zerolog.Ctx(span.Context())

	var unescapedAlias string
	var unescapeErr error
	unescapedAlias, unescapeErr = url.PathUnescape(alias)
	if unescapeErr == nil {
		alias = unescapedAlias
	}

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
func (s *Server) GetClientRoomSummary(ctx context.Context, aliasOrID string) (int, []byte) {
	span := utils.StartSpan(ctx, "matrix.GetClientRoomSummary")
	defer span.Finish()
	log := zerolog.Ctx(span.Context())

	var unescapedAliasOrID string
	var unescapeErr error
	unescapedAliasOrID, unescapeErr = url.PathUnescape(aliasOrID)
	if unescapeErr == nil {
		aliasOrID = unescapedAliasOrID
	}

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
func (s *Server) GetClientRoomVisibility(ctx context.Context, id string) (int, []byte) {
	span := utils.StartSpan(ctx, "matrix.GetClientRoomVisibility")
	defer span.Finish()
	log := zerolog.Ctx(span.Context())

	var unescapedID string
	var unescapeErr error
	unescapedID, unescapeErr = url.PathUnescape(id)
	if unescapeErr == nil {
		id = unescapedID
	}

	room, err := s.data.GetRoom(ctx, id)
	if err != nil {
		log.Error().Err(err).Str("room", id).Msg("cannot get room")
		return http.StatusInternalServerError, s.getErrorResp(span.Context(), "M_INTERNAL_ERROR", "internal error")
	}
	if room == nil {
		return http.StatusNotFound, s.getErrorResp(span.Context(), "M_NOT_FOUND", "room not found")
	}

	resp, err := utils.JSON(map[string]string{"visibility": "public"}) // MRS doesn't have any other
	if err != nil {
		log.Error().Err(err).Str("room", id).Msg("cannot marshal room visibility")
		return http.StatusInternalServerError, s.getErrorResp(span.Context(), "M_INTERNAL_ERROR", "internal error")
	}
	return http.StatusOK, resp
}

// GetClientMediaThumbnail is /_matrix/media/v3/thumbnail/{serverName}/{mediaID}
func (s *Server) GetClientMediaThumbnail(ctx context.Context, serverName, mediaID string, params url.Values) (io.Reader, string) {
	span := utils.StartSpan(ctx, "matrix.GetClientMediaThumbnail")
	defer span.Finish()

	query := utils.ValuesOrDefault(params, defaultThumbnailParams)
	urls := make([]string, 0, len(mediaFallbacks)+1)
	serverURL := s.QueryCSURL(span.Context(), serverName)
	if serverURL != "" {
		urls = append(urls, serverURL+"/_matrix/media/v3/thumbnail/"+serverName+"/"+mediaID+"?"+query)
	}
	for _, serverURL := range mediaFallbacks {
		urls = append(urls, serverURL+"/_matrix/media/v3/thumbnail/"+serverName+"/"+mediaID+"?"+query)
	}
	for _, avatarURL := range urls {
		resp, err := utils.Get(span.Context(), avatarURL)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		return resp.Body, resp.Header.Get("Content-Type")
	}

	return nil, ""
}
