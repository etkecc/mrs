package matrix

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
	"github.com/etkecc/mrs/internal/version"
)

var defaultThumbnailParams = url.Values{
	"animated": []string{"true"},
	"width":    []string{"40"},
	"height":   []string{"40"},
	"method":   []string{"crop"},
}.Encode()

// GetMediaThumbnail is /_matrix/federation/v1/media/thumbnail/{mediaId}
func (s *Server) GetMediaThumbnail(ctx context.Context, serverName, mediaID string, params url.Values) (content io.Reader, contentType string) {
	span := utils.StartSpan(ctx, "matrix.GetMediaThumbnail")
	defer span.Finish()
	log := zerolog.Ctx(span.Context())

	serverURL := s.QueryCSURL(span.Context(), serverName)
	if serverURL == "" {
		log.Warn().Str("server", serverName).Msg("cannot get CS URL")
		return nil, ""
	}

	query := utils.ValuesOrDefault(params, defaultThumbnailParams)
	path := "/_matrix/federation/v1/media/thumbnail/" + mediaID
	apiURL := serverURL + path + "?" + query
	authHeaders, err := s.Authorize(serverName, http.MethodGet, path+"?"+query, nil)
	if err != nil {
		log.Warn().Err(err).Str("server", serverName).Str("mediaID", mediaID).Msg("cannot authorize")
		return nil, ""
	}

	ctx, cancel := context.WithTimeout(span.Context(), utils.DefaultTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		log.Warn().Err(err).Str("server", serverName).Str("mediaID", mediaID).Msg("cannot create request")
		return nil, ""
	}
	for _, h := range authHeaders {
		req.Header.Add("Authorization", h)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", version.UserAgent)

	resp, err := utils.Do(req)
	if err != nil {
		log.Warn().Err(err).Str("server", serverName).Str("mediaID", mediaID).Msg("cannot get media thumbnail")
		return nil, ""
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // intended
		resp.Body.Close()
		log.Warn().Str("server", serverName).Str("mediaID", mediaID).Int("status", resp.StatusCode).Str("body", string(body)).Msg("cannot get media thumbnail")
		return nil, ""
	}
	return s.getImageFromMultipart(span.Context(), resp)
}

// QueryServerName finds server name on the /_matrix/key/v2/server page
func (s *Server) QueryServerName(ctx context.Context, serverName string) (string, error) {
	span := utils.StartSpan(ctx, "matrix.QueryServerName")
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
func (s *Server) QueryDirectory(ctx context.Context, req *http.Request, alias string) (statusCode int, respb []byte) {
	span := utils.StartSpan(ctx, "matrix.QueryDirectory")
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
	respb, err = utils.JSON(resp)
	if err != nil {
		log.Error().Err(err).Msg("cannot marshal query directory resp")
		return http.StatusInternalServerError, nil
	}

	return http.StatusOK, respb
}

// QueryVersion from /_matrix/federation/v1/version
func (s *Server) QueryVersion(ctx context.Context, serverName string) (server, serverVersion string, err error) {
	span := utils.StartSpan(ctx, "matrix.QueryVersion")
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
	span := utils.StartSpan(ctx, "matrix.QueryPublicRooms")
	defer span.Finish()

	ctx, cancel := context.WithTimeout(span.Context(), utils.DefaultTimeout)
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
				bodyhint = fmt.Sprintf("; body: %s", utils.Truncate(string(body), 400))
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

// QueryCSURL returns URL of Matrix CS API server
func (s *Server) QueryCSURL(ctx context.Context, serverName string) string {
	cached, ok := s.curlsCache.Get(serverName)
	if ok {
		return cached
	}

	span := utils.StartSpan(ctx, "matrix.QueryCSURL")
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), http.NoBody)
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

func (s *Server) getImageFromMultipart(ctx context.Context, resp *http.Response) (contentStream io.Reader, contentType string) {
	log := zerolog.Ctx(ctx)
	_, mediaParams, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil {
		log.Warn().Err(err).Msg("cannot parse content type")
		return nil, ""
	}
	mr := multipart.NewReader(resp.Body, mediaParams["boundary"])
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Warn().Err(err).Msg("cannot read multipart")
			return nil, ""
		}
		if p.Header.Get("Content-Type") == "application/json" {
			continue
		}
		if strings.HasPrefix(p.Header.Get("Content-Type"), "image/") {
			return p, p.Header.Get("Content-Type")
		}
	}
	log.Warn().Msg("cannot find image in multipart")
	return nil, ""
}

// trackSearch is a helper function to track search events
func (s *Server) trackSearch(ctx context.Context, req *http.Request, origin, ip, query string) {
	if req.Referer() == "" {
		req.Header.Set("Referer", "https://"+origin)
	}
	// hacky workaround to signal plausible that this is a search via Matrix Federation.
	// Plausible uses https://github.com/matomo-org/device-detector library to parse user-agents,
	// and if the user-agent is empty (or doesn't match any known user-agent for that library),
	// it will not recognize it.
	// To avoid such issues, we set the user-agent to "Synapse" if it is empty,
	// knowing well that this is not a real user-agent, but at least it will be recognized.
	if req.UserAgent() == "" {
		req.Header.Set("User-Agent", "Synapse")
	}
	s.plausible.TrackSearch(ctx, req, ip, query)
}
