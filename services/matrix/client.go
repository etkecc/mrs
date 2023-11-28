package matrix

import (
	"io"
	"net/http"
	"net/url"

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
func (s *Server) GetClientDirectory(alias string) (int, []byte) {
	var unescapedAlias string
	var unescapeErr error
	unescapedAlias, unescapeErr = url.PathUnescape(alias)
	if unescapeErr == nil {
		alias = unescapedAlias
	}

	utils.Logger.Info().Str("alias", alias).Str("origin", "client").Msg("querying directory")
	if alias == "" {
		return http.StatusBadRequest, s.getErrorResp("M_INVALID_PARAM", "Room alias invalid")
	}

	var room *model.MatrixRoom
	s.data.EachRoom(func(_ string, data *model.MatrixRoom) bool {
		if data.Alias == alias {
			room = data
			return true
		}
		return false
	})
	if room == nil {
		return http.StatusNotFound, s.getErrorResp("M_NOT_FOUND", "room not found")
	}

	resp := &queryDirectoryResp{
		RoomID:  room.ID,
		Servers: room.Servers(s.cfg.Get().Matrix.ServerName),
	}
	respb, err := utils.JSON(resp)
	if err != nil {
		utils.Logger.Error().Err(err).Msg("cannot marshal query directory resp")
		return http.StatusInternalServerError, nil
	}

	return http.StatusOK, respb
}

// GetClientRoomSummary is /_matrix/client/unstable/is.nheko.summary/summary/{roomIdOrAlias}
func (s *Server) GetClientRoomSummary(aliasOrID string) (int, []byte) {
	var unescapedAliasOrID string
	var unescapeErr error
	unescapedAliasOrID, unescapeErr = url.PathUnescape(aliasOrID)
	if unescapeErr == nil {
		aliasOrID = unescapedAliasOrID
	}

	utils.Logger.Info().Str("aliasOrID", aliasOrID).Str("origin", "client").Msg("getting room summary")
	if aliasOrID == "" {
		return http.StatusBadRequest, s.getErrorResp("M_INVALID_PARAM", "Room alias or id is invalid")
	}

	var room *model.MatrixRoom
	s.data.EachRoom(func(_ string, data *model.MatrixRoom) bool {
		if data.Alias == aliasOrID || data.ID == aliasOrID {
			room = data
			return true
		}
		return false
	})
	if room == nil {
		return http.StatusNotFound, s.getErrorResp("M_NOT_FOUND", "room not found")
	}
	respb, err := utils.JSON(room.DirectoryEntry())
	if err != nil {
		utils.Logger.Error().Err(err).Msg("cannot marshal room into room directory entry")
		return http.StatusInternalServerError, nil
	}

	return http.StatusOK, respb
}

// GetClientRoomVisibility is /_matrix/client/v3/directory/list/room/{roomID}
func (s *Server) GetClientRoomVisibility(id string) (int, []byte) {
	var unescapedID string
	var unescapeErr error
	unescapedID, unescapeErr = url.PathUnescape(id)
	if unescapeErr == nil {
		id = unescapedID
	}

	room, err := s.data.GetRoom(id)
	if err != nil {
		utils.Logger.Error().Err(err).Str("room", id).Msg("cannot get room")
		return http.StatusInternalServerError, s.getErrorResp("M_INTERNAL_ERROR", "internal error")
	}
	if room == nil {
		return http.StatusNotFound, s.getErrorResp("M_NOT_FOUND", "room not found")
	}

	resp, err := utils.JSON(map[string]string{"visibility": "public"}) // MRS doesn't have any other
	if err != nil {
		utils.Logger.Error().Err(err).Str("room", id).Msg("cannot marshal room visibility")
		return http.StatusInternalServerError, s.getErrorResp("M_INTERNAL_ERROR", "internal error")
	}
	return http.StatusOK, resp
}

// GetClientMediaThumbnail is /_matrix/media/v3/thumbnail/{serverName}/{mediaID}
func (s *Server) GetClientMediaThumbnail(serverName, mediaID string, params url.Values) (io.Reader, string) {
	query := utils.ValuesOrDefault(params, defaultThumbnailParams)
	urls := make([]string, 0, len(mediaFallbacks)+1)
	for _, serverURL := range mediaFallbacks {
		urls = append(urls, serverURL+"/_matrix/media/v3/thumbnail/"+serverName+"/"+mediaID+"?"+query)
	}
	serverURL := s.QueryCSURL(serverName)
	if serverURL != "" {
		urls = append(urls, serverURL+"/_matrix/media/v3/thumbnail/"+serverName+"/"+mediaID+"?"+query)
	}
	datachan := make(chan map[string]io.ReadCloser, 1)
	for _, avatarURL := range urls {
		go downloadThumbnail(datachan, avatarURL)
	}

	for contentType, avatar := range <-datachan {
		close(datachan)
		return avatar, contentType
	}

	return nil, ""
}

func downloadThumbnail(datachan chan map[string]io.ReadCloser, avatarURL string) {
	defer func() {
		if r := recover(); r != nil {
			utils.Logger.Warn().Interface("panic", r).Msg("panic in downloadThumbnail")
		}
	}()

	select {
	case <-datachan:
		return
	default:
		resp, err := http.Get(avatarURL)
		if err != nil {
			return
		}
		if resp.StatusCode != http.StatusOK {
			return
		}
		datachan <- map[string]io.ReadCloser{
			resp.Header.Get("Content-Type"): resp.Body,
		}
	}
}
