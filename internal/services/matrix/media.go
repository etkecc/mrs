package matrix

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/etkecc/go-apm"

	"github.com/etkecc/mrs/internal/utils"
	"github.com/etkecc/mrs/internal/version"
)

// GetMediaThumbnail is /_matrix/federation/v1/media/thumbnail/{mediaId}
func (s *Server) GetMediaThumbnail(ctx context.Context, serverName, mediaID string, params url.Values) (content io.Reader, contentType string) {
	log := apm.Log(ctx)

	params = utils.ValuesOrDefault(params, s.getDefaultThumbnailParams())
	if content, contentType := s.media.Get(ctx, serverName, mediaID, params); content != nil && contentType != "" {
		return content, contentType
	}

	serverURL := s.getURL(ctx, serverName, false)
	if serverURL == "" {
		log.Warn().Str("server", serverName).Msg("cannot get server URL")
		return nil, ""
	}

	query := params.Encode()
	path := "/_matrix/federation/v1/media/thumbnail/" + mediaID
	apiURL := serverURL + path + "?" + query
	authHeaders, err := s.Authorize(serverName, http.MethodGet, path+"?"+query, nil)
	if err != nil {
		log.Warn().Err(err).Str("server", serverName).Str("mediaID", mediaID).Msg("cannot authorize")
		return nil, ""
	}

	ctx, cancel := context.WithTimeout(ctx, utils.DefaultTimeout)
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
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // intended
		resp.Body.Close()
		log.Warn().Str("server", serverName).Str("mediaID", mediaID).Int("status", resp.StatusCode).Str("body", string(body)).Msg("cannot get media thumbnail")
		return nil, ""
	}
	reader, contentType := s.getImageFromMultipart(ctx, resp)
	if reader == nil {
		return nil, ""
	}
	newReader, contents := s.readerBytes(reader)
	s.media.Add(ctx, serverName, mediaID, params, contents)
	return newReader, contentType
}

// GetClientMediaThumbnail is /_matrix/media/v3/thumbnail/{serverName}/{mediaID}
//
// Deprecated: use GetMediaThumbnail() instead, ref: https://spec.matrix.org/v1.11/server-server-api/#get_matrixfederationv1mediathumbnailmediaid
func (s *Server) GetClientMediaThumbnail(ctx context.Context, serverName, mediaID string, params url.Values) (content io.Reader, contentType string) {
	params = utils.ValuesOrDefault(params, s.getDefaultThumbnailParams())
	if content, contentType := s.media.Get(ctx, serverName, mediaID, params); content != nil && contentType != "" {
		return content, contentType
	}

	query := params.Encode()
	urls := make([]string, 0, len(fallbacks)+1)
	serverURL := s.QueryCSURL(ctx, serverName)
	if serverURL != "" {
		urls = append(urls, serverURL+"/_matrix/media/v3/thumbnail/"+serverName+"/"+mediaID+"?"+query)
	}
	for _, serverURL := range fallbacks {
		urls = append(urls, serverURL+"/_matrix/media/v3/thumbnail/"+serverName+"/"+mediaID+"?"+query)
	}
	for _, avatarURL := range urls {
		resp, err := utils.Get(ctx, avatarURL)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		contentType := resp.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "image/") {
			reader, contents := s.readerBytes(resp.Body)
			resp.Body.Close()
			s.media.Add(ctx, serverName, mediaID, params, contents)
			return reader, contentType
		}
		return resp.Body, resp.Header.Get("Content-Type")
	}

	return nil, ""
}

// readerBytes reads the contents of a reader and returns a new reader and the contents
func (s *Server) readerBytes(reader io.Reader) (newReader io.Reader, contents []byte) {
	contents, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil
	}
	newReader = bytes.NewReader(contents)
	return newReader, contents
}

// getImageFromMultipart reads the image from the multipart response
func (s *Server) getImageFromMultipart(ctx context.Context, resp *http.Response) (contentStream io.Reader, contentType string) {
	log := apm.Log(ctx)
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

// getDefaultThumbnailParams returns the default thumbnail parameters
// it is intentionally returning a new url.Values object to avoid concurrent map access
func (s *Server) getDefaultThumbnailParams() url.Values {
	return url.Values{
		"animated": []string{"true"},
		"width":    []string{"40"},
		"height":   []string{"40"},
		"method":   []string{"crop"},
	}
}
