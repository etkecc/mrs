package matrix

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/etkecc/go-apm"
	"github.com/goccy/go-json"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

// getErrorResp returns canonical json of matrix error
func (s *Server) getErrorResp(ctx context.Context, code, message string) []byte {
	respb, err := utils.JSON(model.MatrixError{
		Code:    code,
		Message: message,
	})
	if err != nil {
		apm.Log(ctx).Error().Err(err).Msg("cannot marshal canonical json")
	}
	return respb
}

func (s *Server) parseErrorResp(status string, body []byte) *model.MatrixError {
	if len(body) == 0 {
		return nil
	}
	var merr *model.MatrixError
	if err := json.Unmarshal(body, &merr); err != nil {
		return nil
	}
	if merr.Code == "" {
		return nil
	}

	merr.HTTP = status
	return merr
}

// parseClientWellKnown returns URL of the Matrix CS API server
func (s *Server) parseClientWellKnown(ctx context.Context, serverName string) (string, error) {
	resp, err := utils.Get(ctx, "https://"+serverName+"/.well-known/matrix/client")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("no /.well-known/matrix/client")
	}

	datab, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var wellknown *wellKnownClientResp
	if wkerr := json.Unmarshal(datab, &wellknown); wkerr != nil {
		return "", wkerr
	}
	if wellknown.Homeserver.BaseURL == "" {
		return "", fmt.Errorf("/.well-known/matrix/client is empty")
	}

	// fixing misconfigured well-known, example.com:443 -> https://example.com
	if !strings.HasPrefix(wellknown.Homeserver.BaseURL, "https://") && !strings.HasPrefix(wellknown.Homeserver.BaseURL, "http://") {
		wellknown.Homeserver.BaseURL = "https://" + wellknown.Homeserver.BaseURL
	}
	wellknown.Homeserver.BaseURL = strings.TrimSuffix(wellknown.Homeserver.BaseURL, ":443")
	return wellknown.Homeserver.BaseURL, nil
}

// parseServerWellKnown returns Federation API host:port
func (s *Server) parseServerWellKnown(ctx context.Context, serverName string) (string, error) {
	log := apm.Log(ctx).With().Str("server", serverName).Logger()

	resp, err := utils.Get(ctx, "https://"+serverName+"/.well-known/matrix/server")
	if err != nil {
		log.Warn().Err(err).Msg("failed to get /.well-known/matrix/server")
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Warn().Str("status", resp.Status).Msg("no /.well-known/matrix/server")
		return "", fmt.Errorf("no /.well-known/matrix/server")
	}

	datab, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warn().Err(err).Msg("failed to read /.well-known/matrix/server")
		return "", err
	}
	var wellknown *wellKnownServerResp
	if wkerr := json.Unmarshal(datab, &wellknown); wkerr != nil {
		log.Warn().Err(wkerr).Msg("failed to parse /.well-known/matrix/server")
		return "", wkerr
	}
	if wellknown.Host == "" {
		log.Warn().Msg("/.well-known/matrix/server is empty")
		return "", fmt.Errorf("/.well-known/matrix/server is empty")
	}
	parts := strings.Split(wellknown.Host, ":")
	if len(parts) == 0 {
		log.Warn().Msg("/.well-known/matrix/server is invalid")
		return "", fmt.Errorf("/.well-known/matrix/server is invalid")
	}
	host := parts[0]
	port := "8448"
	if len(parts) == 2 {
		port = parts[1]
	}
	return host + ":" + port, err
}

// parseSRV returns Federation API host:port
func (s *Server) parseSRV(ctx context.Context, service, serverName string) (string, error) {
	_, addrs, err := net.DefaultResolver.LookupSRV(ctx, service, "tcp", serverName)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", errors.New("no " + service + " SRV records")
	}
	return strings.Trim(addrs[0].Target, ".") + ":" + strconv.Itoa(int(addrs[0].Port)), nil
}

// dcrURL stands for discover-cache-and-return URL, shortcut for s.getURL
func (s *Server) dcrURL(ctx context.Context, serverName, serverURL string, discover bool) string {
	s.surlsCache.Add(serverName, serverURL)

	if s.discoverFunc != nil && discover {
		go s.discoverFunc(ctx, serverName)
	}

	return serverURL
}

// getURL returns Federation API URL
func (s *Server) getURL(ctx context.Context, serverName string, discover bool) string {
	cached, ok := s.surlsCache.Get(serverName)
	if ok {
		return cached
	}

	log := apm.Log(ctx).With().Str("server", serverName).Logger()
	fromWellKnown, err := s.parseServerWellKnown(ctx, serverName)
	if err == nil {
		return s.dcrURL(ctx, serverName, "https://"+fromWellKnown, discover)
	}
	log.Warn().Err(err).Msg("failed to parse /.well-known/matrix/server")

	fromSRV, err := s.parseSRV(ctx, "matrix-fed", serverName)
	if err == nil {
		return s.dcrURL(ctx, serverName, "https://"+fromSRV, discover)
	}
	log.Warn().Err(err).Msg("failed to parse SRV matrix-fed")

	fromSRV, err = s.parseSRV(ctx, "matrix", serverName)
	if err == nil {
		return s.dcrURL(ctx, serverName, "https://"+fromSRV, discover)
	}
	log.Warn().Err(err).Msg("failed to parse SRV matrix, using server name as host")

	return s.dcrURL(ctx, serverName, "https://"+serverName+":8448", discover)
}

// lookupKeys requests /_matrix/key/v2/server by serverName
func (s *Server) lookupKeys(ctx context.Context, serverName string, discover bool) (*matrixKeyResp, error) {
	log := apm.Log(ctx).With().Str("server", serverName).Logger()

	keysURL, err := url.Parse(s.getURL(ctx, serverName, discover) + "/_matrix/key/v2/server")
	if err != nil {
		log.Warn().Err(err).Msg("failed to parse keys URL")
		return nil, err
	}
	resp, err := utils.Get(ctx, keysURL.String())
	if err != nil {
		log.Warn().Err(err).Msg("failed to get keys")
		return nil, err
	}
	defer resp.Body.Close()
	datab, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warn().Err(err).Msg("failed to read keys")
		return nil, err
	}
	if merr := s.parseErrorResp(resp.Status, datab); merr != nil {
		log.Warn().Str("http", merr.HTTP).Str("code", merr.Code).Str("message", merr.Message).Msg("keys query failed")
		return nil, merr
	}

	var keysResp *matrixKeyResp
	if err := json.Unmarshal(datab, &keysResp); err != nil {
		log.Warn().Err(err).Msg("failed to parse keys")
		return nil, err
	}
	return keysResp, nil
}

// queryKeys returns serverName's keys
func (s *Server) queryKeys(ctx context.Context, serverName string) map[string]ed25519.PublicKey {
	cached, ok := s.keysCache.Get(serverName)
	if ok {
		return cached
	}
	log := apm.Log(ctx)
	resp, err := s.lookupKeys(ctx, serverName, true)
	if err != nil {
		log.Warn().Err(err).Msg("keys query failed")
		return nil
	}
	if resp.ServerName != serverName {
		log.Warn().Msg("server name doesn't match")
		return nil
	}
	if resp.ValidUntilTS <= time.Now().UnixMilli() {
		log.Warn().Msg("server keys are expired")
	}
	keys := map[string]ed25519.PublicKey{}
	for id, data := range resp.VerifyKeys {
		pub, err := base64.RawStdEncoding.DecodeString(data["key"])
		if err != nil {
			log.Warn().Err(err).Msg("failed to decode server key")
			continue
		}
		keys[id] = pub
	}
	// TODO: verify signatures
	s.keysCache.Add(serverName, keys)
	return keys
}
