package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"

	"github.com/etkecc/mrs/internal/utils"
)

// Media service controls (cached) avatars files on disk
type Media struct {
	cfg  ConfigService
	base *url.URL
}

type MediaService interface {
	Exists(serverName, mediaID string, params url.Values) bool
	GetURL(serverName, mediaID string) string
	Get(ctx context.Context, serverName, mediaID string, params url.Values) (content io.Reader, contentType string)
	Add(ctx context.Context, serverName, mediaID string, params url.Values, content []byte)
	Delete(ctx context.Context, serverName, mediaID string)
}

var _ MediaService = (*Media)(nil)

func NewMedia(cfg ConfigService) (*Media, error) {
	m := &Media{
		cfg: cfg,
	}
	url, err := url.Parse(cfg.Get().Public.API)
	if err != nil {
		return nil, fmt.Errorf("invalid public API URL: %w", err)
	}
	m.base = url
	return m, nil
}

// Exists checks if the media file exists on disk
func (m *Media) Exists(serverName, mediaID string, params url.Values) bool {
	path := m.getPath(serverName, mediaID, params)
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// GetURL returns the URL of the media file
func (m *Media) GetURL(serverName, mediaID string) string {
	return m.base.JoinPath("/avatar", serverName, mediaID).String()
}

// Get retrieves the media file from disk
func (m *Media) Get(ctx context.Context, serverName, mediaID string, params url.Values) (content io.Reader, contentType string) {
	if !m.Exists(serverName, mediaID, params) {
		return nil, ""
	}

	mediaPath := m.getPath(serverName, mediaID, params)
	if mediaPath == "" {
		return nil, ""
	}

	log := zerolog.Ctx(ctx).With().Str("server", serverName).Str("mediaID", mediaID).Str("file", mediaPath).Logger()
	file, err := os.Open(mediaPath)
	if err != nil {
		log.Warn().Err(err).Msg("cannot open media file")
		return nil, ""
	}
	defer file.Close()

	// detect content type
	sniff := make([]byte, 512)
	if _, err := file.Read(sniff); err != nil {
		log.Warn().Msg("cannot read media file")
		return nil, ""
	}
	contentType = http.DetectContentType(sniff)

	if _, err := file.Seek(0, 0); err != nil {
		log.Warn().Err(err).Msg("cannot seek media file")
		return nil, ""
	}
	contents, err := io.ReadAll(file)
	if err != nil {
		log.Warn().Err(err).Msg("cannot read media file")
		return nil, ""
	}
	return bytes.NewReader(contents), contentType
}

// Add saves the media file to disk if it doesn't already exist
func (m *Media) Add(ctx context.Context, serverName, mediaID string, params url.Values, content []byte) {
	mediaPath := m.getPath(serverName, mediaID, params)
	if mediaPath == "" {
		return
	}

	// Matrix media is immutable, so if the file already exists, we don't need to write it again
	if _, err := os.Stat(mediaPath); !os.IsNotExist(err) {
		return
	}

	log := zerolog.Ctx(ctx).With().Str("server", serverName).Str("mediaID", mediaID).Str("file", mediaPath).Logger()
	if err := os.MkdirAll(filepath.Dir(mediaPath), 0o755); err != nil {
		log.Warn().Err(err).Msg("cannot create media directory")
		return
	}

	if err := os.WriteFile(mediaPath, content, 0o600); err != nil {
		log.Warn().Err(err).Msg("cannot write media file")
	}
}

// Delete removes the media file from disk
func (m *Media) Delete(ctx context.Context, serverName, mediaID string) {
	media := m.cfg.Get().Path.Media
	log := zerolog.Ctx(ctx).With().Str("server", serverName).Str("mediaID", mediaID).Logger()
	entries, err := os.ReadDir(media)
	if err != nil {
		log.Warn().Err(err).Msg("cannot read media directory")
		return
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), fmt.Sprintf("%s-%s-", serverName, mediaID)) {
			continue
		}
		if err := os.Remove(filepath.Join(media, entry.Name())); err != nil {
			log.Warn().Str("file", entry.Name()).Err(err).Msg("cannot delete media file")
		}
	}
}

// getPath returns the path to the media file on disk
func (m *Media) getPath(serverName, mediaID string, params url.Values) string {
	mediaPath := m.cfg.Get().Path.Media
	if mediaPath == "" || serverName == "" || mediaID == "" {
		return ""
	}
	var filename strings.Builder
	filename.WriteString(serverName)
	filename.WriteString("-")
	filename.WriteString(mediaID)
	filename.WriteString("-")
	if len(params) > 0 {
		filename.WriteString(utils.HashURLValues(params))
	}
	return filepath.Join(mediaPath, filename.String())
}
