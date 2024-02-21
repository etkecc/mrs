package matrix

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/getsentry/sentry-go"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
	"gitlab.com/etke.cc/mrs/api/version"
)

// QueryServerName finds server name on the /_matrix/key/v2/server page
func (s *Server) QueryServerName(ctx context.Context, serverName string) (string, error) {
	span := sentry.StartSpan(ctx, "function", sentry.WithDescription("matrix.QueryServerName"))
	defer span.Finish()
	log := zerolog.Ctx(span.Context())

	cached, ok := s.namesCache.Get(serverName)
	if ok {
		return cached, nil
	}
	discovered := ""
	resp, err := s.lookupKeys(span.Context(), serverName, false)
	if err == nil && resp != nil {
		discovered = resp.ServerName
		s.namesCache.Add(serverName, discovered)
	} else {
		log.Warn().Err(err).Str("server", serverName).Msg("cannot query server name")
	}

	return discovered, err
}

// QueryDirectory is /_matrix/federation/v1/query/directory?room_alias={roomAlias}
func (s *Server) QueryDirectory(ctx context.Context, req *http.Request, alias string) (int, []byte) {
	span := sentry.StartSpan(ctx, "function", sentry.WithDescription("matrix.QueryDirectory"))
	defer span.Finish()
	log := zerolog.Ctx(span.Context())

	origin, err := s.ValidateAuth(span.Context(), req)
	if err != nil {
		log.Warn().Err(err).Msg("matrix auth failed")
		return http.StatusUnauthorized, s.getErrorResp(span.Context(), "M_UNAUTHORIZED", "authorization failed")
	}

	var unescapedAlias string
	var unescapeErr error
	unescapedAlias, unescapeErr = url.QueryUnescape(alias)
	if unescapeErr == nil {
		alias = unescapedAlias
	}
	log.Info().Str("alias", alias).Str("origin", origin).Msg("querying directory")
	if alias == "" {
		return http.StatusNotFound, s.getErrorResp(span.Context(), "M_NOT_FOUND", "room not found")
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

// QueryVersion from /_matrix/federation/v1/version
func (s *Server) QueryVersion(ctx context.Context, serverName string) (server, version string, err error) {
	span := sentry.StartSpan(ctx, "function", sentry.WithDescription("matrix.QueryVersion"))
	defer span.Finish()

	resp, err := utils.Get(span.Context(), s.getURL(span.Context(), serverName, false)+"/_matrix/federation/v1/version")
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
	span := sentry.StartSpan(ctx, "function", sentry.WithDescription("matrix.QueryPublicRooms"))
	defer span.Finish()

	ctx, cancel := context.WithTimeout(span.Context(), utils.DefaultTimeout)
	defer cancel()
	req, err := s.buildPublicRoomsReq(ctx, serverName, limit, since)
	if err != nil {
		return nil, err
	}

	resp, err := utils.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // intended
		merr := s.parseErrorResp(resp.Status, body)
		if merr == nil {
			return nil, fmt.Errorf("cannot get public rooms: %s", resp.Status)
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

// QueryCSURL returns URL of Matrix CS API server
func (s *Server) QueryCSURL(ctx context.Context, serverName string) string {
	cached, ok := s.curlsCache.Get(serverName)
	if ok {
		return cached
	}

	span := sentry.StartSpan(ctx, "function", sentry.WithDescription("matrix.QueryCSURL"))
	defer span.Finish()

	csurl := "https://" + serverName
	fromWellKnown, err := s.parseClientWellKnown(ctx, serverName)
	if err == nil {
		csurl = fromWellKnown
	}

	s.curlsCache.Add(serverName, csurl)
	return csurl
}

func (s *Server) buildPublicRoomsReq(ctx context.Context, serverName, limit, since string) (*http.Request, error) {
	apiURLStr := s.getURL(ctx, serverName, false)
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), nil)
	if err != nil {
		return nil, err
	}
	for _, h := range authHeaders {
		req.Header.Add("Authorization", h)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", version.UserAgent)
	return req, nil
}
