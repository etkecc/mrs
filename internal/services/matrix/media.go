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

	"github.com/etkecc/mrs/internal/utils"
	"github.com/etkecc/mrs/internal/version"
	"github.com/rs/zerolog"
)

var defaultThumbnailParams = url.Values{
	"animated": []string{"true"},
	"width":    []string{"40"},
	"height":   []string{"40"},
	"method":   []string{"crop"},
}

// GetMediaThumbnail is /_matrix/federation/v1/media/thumbnail/{mediaId}
func (s *Server) GetMediaThumbnail(ctx context.Context, serverName, mediaID string, params url.Values) (content io.Reader, contentType string) {
	span := utils.StartSpan(ctx, "matrix.GetMediaThumbnail")
	defer span.Finish()
	log := zerolog.Ctx(span.Context())

	params = utils.ValuesOrDefault(params, defaultThumbnailParams)
	if content, contentType := s.media.Get(span.Context(), serverName, mediaID, params); content != nil {
		return content, contentType
	}

	serverURL := s.getURL(span.Context(), serverName, false)
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

	resp, err := utils.Do(req, 0)
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
	reader, contentType := s.getImageFromMultipart(span.Context(), resp)
	if reader == nil {
		return nil, ""
	}
	newReader, contents := s.readerBytes(reader)
	s.media.Add(span.Context(), serverName, mediaID, params, contents)
	return newReader, contentType
}

// GetClientMediaThumbnail is /_matrix/media/v3/thumbnail/{serverName}/{mediaID}
// Deprecated: use GetMediaThumbnail() instead, ref: https://spec.matrix.org/v1.11/server-server-api/#get_matrixfederationv1mediathumbnailmediaid
func (s *Server) GetClientMediaThumbnail(ctx context.Context, serverName, mediaID string, params url.Values) (content io.Reader, contentType string) {
	span := utils.StartSpan(ctx, "matrix.GetClientMediaThumbnail")
	defer span.Finish()

	params = utils.ValuesOrDefault(params, defaultThumbnailParams)
	if content, contentType := s.media.Get(span.Context(), serverName, mediaID, params); content != nil {
		return content, contentType
	}

	query := params.Encode()
	urls := make([]string, 0, len(mediaFallbacks)+1)
	serverURL := s.QueryCSURL(span.Context(), serverName)
	if serverURL != "" {
		urls = append(urls, serverURL+"/_matrix/media/v3/thumbnail/"+serverName+"/"+mediaID+"?"+query)
	}
	for _, serverURL := range mediaFallbacks {
		urls = append(urls, serverURL+"/_matrix/media/v3/thumbnail/"+serverName+"/"+mediaID+"?"+query)
	}
	for _, avatarURL := range urls {
		resp, err := utils.Get(span.Context(), avatarURL, 0)
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
			s.media.Add(span.Context(), serverName, mediaID, params, contents)
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
