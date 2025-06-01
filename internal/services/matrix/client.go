package matrix

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/etkecc/go-apm"
	"github.com/etkecc/go-kit"
	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
	"github.com/goccy/go-json"
)

var (
	fallbacks      = []string{"https://matrix-client.matrix.org"}
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
		Servers: room.Servers(),
	}
	respb, err = utils.JSON(resp)
	if err != nil {
		log.Error().Err(err).Msg("cannot marshal query directory resp")
		return http.StatusInternalServerError, nil
	}

	return http.StatusOK, respb
}

// GetClientRoomSummary is /_matrix/client/unstable/is.nheko.summary/summary/{roomIdOrAlias}
func (s *Server) GetClientRoomSummary(ctx context.Context, aliasOrID, via string, onlyMSC3266 bool) (statusCode int, directoryRoom *model.RoomDirectoryRoom) {
	log := apm.Log(ctx)
	aliasOrID = utils.Unescape(aliasOrID)
	log.Info().Str("aliasOrID", aliasOrID).Str("origin", "client").Msg("getting room summary")
	if aliasOrID == "" {
		return http.StatusBadRequest, nil
	}
	var entry *model.RoomDirectoryRoom
	// 1. Try to get the room from the database
	if !onlyMSC3266 {
		entry = s.roomSummaryDirect(ctx, aliasOrID)
	}

	// 2. If the room is not found (or only MSC3266 is allowed), try to get it from the fallback (or via GET param)
	if entry == nil {
		// Fallback to MSC3266 if room is not found in the database
		entry = s.roomSummaryFallback(ctx, aliasOrID, via)
	}

	// 3. If the room is still not found, return 404
	if entry == nil {
		return http.StatusNotFound, nil
	}

	if s.data.IsBanned(ctx, entry.ID) {
		log.Warn().Msg("attempting to get summary of a banned room")
		return http.StatusNotFound, nil
	}

	servers := kit.Uniq([]string{
		utils.ServerFrom(entry.ID),
		utils.ServerFrom(entry.Alias),
	})

	for _, server := range servers {
		if s.blocklist.ByServer(server) {
			return http.StatusNotFound, nil
		}
	}

	return http.StatusOK, entry
}

// GetClientRoomVisibility is /_matrix/client/v3/directory/list/room/{roomID}
// this is a stub endpoint, because MRS works only with public rooms,
// so we always return public visibility.
// That may change in the future (e.g., if protocol adds more visibility types),
// but for now we just return public visibility.
func (s *Server) GetClientRoomVisibility(_ context.Context, _ string) (statusCode int, resp []byte) {
	return http.StatusOK, roomVisibility
}

// roomSummaryDirect retrieves the room summary directly from the database
func (s *Server) roomSummaryDirect(ctx context.Context, aliasOrID string) *model.RoomDirectoryRoom {
	log := apm.Log(ctx).With().Str("room", aliasOrID).Logger()

	// Try to get the room from the database
	room, err := s.getRoom(ctx, aliasOrID)
	if err != nil {
		log.Error().Err(err).Msg("cannot get room from data store")
		return nil
	}
	if room == nil {
		log.Warn().Msg("room not found in data store")
		return nil
	}

	return room.DirectoryEntry()
}

// roomSummaryFallback uses MSC3266 to find room summary
func (s *Server) roomSummaryFallback(ctx context.Context, aliasOrID, via string) *model.RoomDirectoryRoom {
	log := apm.Log(ctx).With().Str("room", aliasOrID).Str("via", via).Logger()

	if via == "" {
		for _, fallback := range fallbacks {
			room := s.roomSummaryVia(ctx, aliasOrID, fallback)
			if room != nil {
				return room
			}
		}
		return nil
	}

	csURL := s.QueryCSURL(ctx, via)
	if csURL == "" {
		log.Warn().Msg("no CS URL found for user-defined via")
		return nil
	}

	if room := s.roomSummaryVia(ctx, aliasOrID, csURL); room != nil {
		return room
	}

	return nil
}

func (s *Server) roomSummaryVia(ctx context.Context, aliasOrID, via string) *model.RoomDirectoryRoom {
	log := apm.Log(ctx).With().Str("room", aliasOrID).Str("via", via).Logger()
	endpoint := fmt.Sprintf(
		"%s/_matrix/client/unstable/im.nheko.summary/summary/%s?via=%s",
		via,
		url.PathEscape(aliasOrID),
		utils.ServerFrom(aliasOrID),
	)
	log.Info().Str("endpoint", endpoint).Msg("querying room summary from fallback")
	resp, err := utils.Get(ctx, endpoint)
	if err != nil {
		log.Warn().Err(err).Msg("cannot get room summary from fallback")
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Warn().Int("statusCode", resp.StatusCode).Msg("fallback returned non-OK status code")
		return nil
	}
	var directoryEntry *model.RoomDirectoryRoom
	if err := json.NewDecoder(resp.Body).Decode(&directoryEntry); err != nil {
		log.Warn().Err(err).Msg("cannot decode room directory entry from fallback")
		return nil
	}

	if directoryEntry == nil || directoryEntry.ID == "" {
		log.Warn().Msg("fallback returned empty room directory entry")
		return nil
	}

	return directoryEntry
}
