package matrix

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/etkecc/go-apm"
	"github.com/etkecc/go-kit"
	"github.com/etkecc/go-kit/workpool"
	"github.com/goccy/go-json"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
	"github.com/etkecc/mrs/internal/version"
)

// QueryServerName finds server name on the /_matrix/key/v2/server page
func (s *Server) QueryServerName(ctx context.Context, serverName string) (string, error) {
	log := apm.Log(ctx)

	cached, ok := s.namesCache.Get(serverName)
	if ok {
		return cached, nil
	}
	discovered := ""
	resp, err := s.lookupKeys(ctx, serverName, false)
	if err == nil && resp != nil {
		discovered = resp.ServerName
		s.namesCache.Add(serverName, discovered)
	} else {
		log.Warn().Err(err).Str("server", serverName).Msg("cannot query server name")
	}

	return discovered, err
}

// QueryDirectory is /_matrix/federation/v1/query/directory?room_alias={roomAlias}
func (s *Server) QueryDirectory(ctx context.Context, req *http.Request, alias string) (statusCode int, respb []byte) {
	log := apm.Log(ctx)

	origin, err := s.ValidateAuth(ctx, req)
	if err != nil {
		log.Warn().Err(err).Msg("matrix auth failed")
		return http.StatusUnauthorized, s.getErrorResp(ctx, "M_UNAUTHORIZED", "authorization failed")
	}

	alias = utils.Unescape(alias)
	log.Info().Str("alias", alias).Str("origin", origin).Msg("querying directory")
	if alias == "" {
		return http.StatusNotFound, s.getErrorResp(ctx, "M_NOT_FOUND", "room not found")
	}
	room, err := s.getRoom(ctx, alias)
	if err != nil {
		log.Error().Err(err).Msg("cannot get room from data store")
	}
	if room == nil {
		return http.StatusNotFound, s.getErrorResp(ctx, "M_NOT_FOUND", "room not found")
	}

	resp := &model.QueryDirectoryResponse{
		RoomID:  room.ID,
		Servers: room.AllServers(),
	}
	respb, err = utils.JSON(resp)
	if err != nil {
		log.Error().Err(err).Msg("cannot marshal query directory resp")
		return http.StatusInternalServerError, nil
	}

	return http.StatusOK, respb
}

// QueryDirectoryExternal queries another server's directory for a room alias
func (s *Server) QueryDirectoryExternal(ctx context.Context, alias string) (*model.QueryDirectoryResponse, error) {
	log := apm.Log(ctx)
	alias = utils.Unescape(alias)
	if alias == "" {
		return nil, fmt.Errorf("room alias invalid")
	}
	log.Info().Str("alias", alias).Msg("querying external directory")
	serverName := utils.ServerFrom(alias)
	if serverName == "" {
		return nil, fmt.Errorf("cannot extract server name from alias")
	}

	ctx, cancel := context.WithTimeout(ctx, utils.DefaultTimeout)
	defer cancel()
	req, err := s.buildQueryDirectoryReq(ctx, serverName, alias)
	if err != nil {
		return nil, err
	}

	resp, err := utils.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // intended
		merr := s.parseErrorResp(resp.Status, body)
		if merr == nil {
			bodyhint := ""
			if len(body) > 0 {
				bodyhint = fmt.Sprintf("; body: %s", kit.Truncate(string(body), 400))
			}
			return nil, fmt.Errorf("cannot query directory: %s%s", resp.Status, bodyhint)
		}
		return nil, merr
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var dirResp *model.QueryDirectoryResponse
	err = json.Unmarshal(data, &dirResp)
	if err != nil {
		return nil, err
	}
	return dirResp, nil
}

// QueryVersion from /_matrix/federation/v1/version
func (s *Server) QueryVersion(ctx context.Context, serverName string) (server, serverVersion string, err error) {
	serverURL, serverHost := s.getURL(ctx, serverName, false)
	resp, err := utils.Get(ctx, serverURL+"/_matrix/federation/v1/version", serverHost)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("federation disabled")
	}

	datab, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	var vResp *serverVersionResp
	if jerr := json.Unmarshal(datab, &vResp); jerr != nil {
		return "", "", jerr
	}
	if len(vResp.Server) == 0 {
		return "", "", fmt.Errorf("invalid version response")
	}
	if vResp.Server["name"] == "" || vResp.Server["version"] == "" {
		return "", "", fmt.Errorf("invalid version contents")
	}

	return vResp.Server["name"], vResp.Server["version"], nil
}

// QueryPublicRooms over federation
func (s *Server) QueryPublicRooms(ctx context.Context, serverName, limit, since string) (*model.RoomDirectoryResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, utils.DefaultTimeout)
	defer cancel()
	req, err := s.buildPublicRoomsReq(ctx, serverName, limit, since)
	if err != nil {
		return nil, err
	}

	resp, err := utils.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // intended
		merr := s.parseErrorResp(resp.Status, body)
		if merr == nil {
			bodyhint := ""
			if len(body) > 0 {
				bodyhint = fmt.Sprintf("; body: %s", kit.Truncate(string(body), 400))
			}
			return nil, fmt.Errorf("cannot get public rooms: %s%s", resp.Status, bodyhint)
		}
		return nil, merr
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var roomsResp *model.RoomDirectoryResponse
	err = json.Unmarshal(data, &roomsResp)
	if err != nil {
		return nil, err
	}
	return roomsResp, nil
}

// QueryServerKeys is /_matrix/key/v2/query/{serverName}
func (s *Server) QueryServerKeys(ctx context.Context, serverName string, validUntilTS int64) []byte {
	log := apm.Log(ctx).With().Str("server", serverName).Logger()
	if validUntilTS == 0 {
		validUntilTS = time.Now().UnixMilli()
	}
	keyPayload := s.notaryLookupKeys(ctx, serverName, validUntilTS)
	if keyPayload == nil {
		return []byte(model.EmptyServerKeysResp)
	}
	payload, err := utils.JSON(matrixKeyQueryResp{
		ServerKeys: []json.RawMessage{keyPayload},
	})
	if err != nil {
		log.Error().Err(err).Msg("cannot marshal key query response")
		return []byte(model.EmptyServerKeysResp)
	}
	return payload
}

// QueryServersKeys is /_matrix/key/v2/query for multiple servers
// Current naive implementation returns all keys, even when request is for specific key IDs
func (s *Server) QueryServersKeys(ctx context.Context, req *model.QueryServerKeysRequest, validUntilTS int64) []byte {
	serverNames := kit.MapKeys(req.ServerKeys)
	log := apm.Log(ctx).With().Strs("servers", serverNames).Logger()
	if validUntilTS == 0 {
		validUntilTS = time.Now().UnixMilli()
	}
	keyPayloads := make([]json.RawMessage, 0, len(serverNames))
	var mu sync.Mutex
	wp := workpool.New(s.cfg.Get().Workers.Discovery)
	for _, serverName := range serverNames {
		wp.Do(func() {
			ctx = context.WithoutCancel(ctx) // cancellation is controlled inside the notaryLookupKeys
			keyPayload := s.notaryLookupKeys(ctx, serverName, validUntilTS)
			if keyPayload != nil {
				mu.Lock()
				keyPayloads = append(keyPayloads, keyPayload)
				mu.Unlock()
			}
		})
	}
	wp.Run()
	payload, err := utils.JSON(matrixKeyQueryResp{
		ServerKeys: keyPayloads,
	})
	if err != nil {
		log.Error().Err(err).Msg("cannot marshal keys query response")
		return []byte(model.EmptyServerKeysResp)
	}
	return payload
}

// QueryCSURL returns URL of Matrix CS API server
func (s *Server) QueryCSURL(ctx context.Context, serverName string) string {
	cached, ok := s.curlsCache.Get(serverName)
	if ok {
		return cached
	}

	csurl := "https://" + serverName
	fromWellKnown, err := s.parseClientWellKnown(ctx, serverName)
	if err == nil {
		csurl = fromWellKnown
	}

	s.curlsCache.Add(serverName, csurl)
	return csurl
}

func (s *Server) buildPublicRoomsReq(ctx context.Context, serverName, limit, since string) (*http.Request, error) {
	apiURLStr, apiURLHost := s.getURL(ctx, serverName, false)
	apiURL, err := url.Parse(apiURLStr)
	if err != nil {
		return nil, err
	}
	apiURL = apiURL.JoinPath("/_matrix/federation/v1/publicRooms")
	query := apiURL.Query()
	if limit != "" {
		query.Set("limit", limit)
	}
	if since != "" {
		query.Set("since", url.QueryEscape(since))
	}
	apiURL.RawQuery = query.Encode()

	path := "/" + apiURL.Path
	if apiURL.RawQuery != "" {
		path += "?" + apiURL.RawQuery
	}
	authHeaders, err := s.Authorize(serverName, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	if apiURLHost != "" {
		req.Host = apiURLHost
	}
	for _, h := range authHeaders {
		req.Header.Add("Authorization", h)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", version.UserAgent)
	return req, nil
}

func (s *Server) buildQueryDirectoryReq(ctx context.Context, serverName, alias string) (*http.Request, error) {
	apiURLStr, apiURLHost := s.getURL(ctx, serverName, false)
	apiURL, err := url.Parse(apiURLStr)
	if err != nil {
		return nil, err
	}
	apiURL = apiURL.JoinPath("/_matrix/federation/v1/query/directory")
	query := apiURL.Query()
	query.Set("room_alias", alias)
	apiURL.RawQuery = query.Encode()

	path := "/" + apiURL.Path
	if apiURL.RawQuery != "" {
		path += "?" + apiURL.RawQuery
	}
	authHeaders, err := s.Authorize(serverName, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	if apiURLHost != "" {
		req.Host = apiURLHost
	}
	for _, h := range authHeaders {
		req.Header.Add("Authorization", h)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", version.UserAgent)
	return req, nil
}

// getRoom is a helper function to get a room by ID or alias
func (s *Server) getRoom(ctx context.Context, roomIDorAlias string) (*model.MatrixRoom, error) {
	roomID := roomIDorAlias
	if utils.IsValidAlias(roomIDorAlias) {
		if mapped := s.data.GetRoomMapping(ctx, roomIDorAlias); mapped != "" {
			roomID = mapped
		}
	}

	return s.data.GetRoom(ctx, roomID)
}
