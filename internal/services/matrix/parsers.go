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
	"github.com/etkecc/go-kit/httpclient"
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

	return wellknown.Host, err
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

// dcrURL stands for discover-cache-and-return URL, shortcut for s.getURL. dialIP is the pre-resolved
// TCP target for a delegated host whose SRV target differs from its name; empty on every other branch.
func (s *Server) dcrURL(ctx context.Context, serverName, serverURL, serverHost, dialIP string, discover bool) (sURL, sHost, sDialIP string) {
	justHost, _, err := net.SplitHostPort(serverHost)
	if err == nil {
		serverHost = justHost
	}

	s.surlsCache.Add(serverName, serverURL+"||"+serverHost+"||"+dialIP)

	// fire-and-forget discovery, bounded by the semaphore: a full pool skips (best-effort, re-triggers later).
	// the ctx detaches from the request so discovery survives it (a pass runs for hours); per-call 120s ceilings bound each hop.
	if s.discoverFunc != nil && discover {
		select {
		case s.discoverSem <- struct{}{}:
			dctx := context.WithoutCancel(ctx)
			go func() {
				defer func() { <-s.discoverSem }()
				s.discoverFunc(dctx, serverName)
			}()
		default:
		}
	}

	return serverURL, serverHost, dialIP
}

// getURL returns the Federation API URL, the delegated Host, and a context pinned to the dial IP when
// the resolved SRV target differs from the delegated host (the sole IP-pin branch).
// Resolution follows https://spec.matrix.org/v1.18/server-server-api/#resolving-server-names
func (s *Server) getURL(ctx context.Context, serverName string, discover bool) (pinnedCtx context.Context, ssURL, ssHost string) {
	if cached, ok := s.surlsCache.Get(serverName); ok {
		parts := strings.Split(cached, "||")
		if len(parts) == 3 {
			return httpclient.WithDialIP(ctx, parts[2]), parts[0], parts[1]
		}
		s.surlsCache.Remove(serverName) // pre-3-part or corrupt entry, drop and re-resolve
	}

	// Step 2: serverName has explicit port, skip well-known and SRV and connect directly.
	// Also covers step 1 (IP literal with port) since net.SplitHostPort handles "[::1]:port".
	if _, _, err := net.SplitHostPort(serverName); err == nil {
		ssURL, ssHost, _ = s.dcrURL(ctx, serverName, "https://"+serverName, serverName, "", discover)
		return ctx, ssURL, ssHost
	}

	// Step 1: bare IP literal with no port, skip well-known and SRV and default to 8448.
	if ip := net.ParseIP(serverName); ip != nil {
		host := serverName
		if ip.To4() == nil {
			host = "[" + serverName + "]"
		}
		ssURL, ssHost, _ = s.dcrURL(ctx, serverName, "https://"+host+":8448", serverName, "", discover)
		return ctx, ssURL, ssHost
	}

	ssURL, ssHost, dialIP := s.getURLFromWK(ctx, serverName, discover)
	if ssURL == "" {
		ssURL, ssHost, dialIP = s.getURLFromSRV(ctx, serverName, discover)
	}
	return httpclient.WithDialIP(ctx, dialIP), ssURL, ssHost
}

// getURLFromSRV tries to get Federation API URL via SRV records.
// It tries _matrix-fed._tcp first, then falls back to legacy _matrix._tcp,
// and finally defaults to port 8448.
func (s *Server) getURLFromSRV(ctx context.Context, serverName string, discover bool) (ssURL, ssHost, dialIP string) {
	log := apm.Log(ctx).With().Str("server", serverName).Logger()
	fromSRV, err := s.parseSRV(ctx, "matrix-fed", serverName)
	if err != nil {
		log.Warn().Err(err).Msg("failed to parse SRV matrix-fed, trying SRV matrix")
		fromSRV, err = s.parseSRV(ctx, "matrix", serverName)
	}
	if err != nil {
		log.Warn().Err(err).Msg("failed to parse SRV matrix, falling back to port 8448")
		return s.dcrURL(ctx, serverName, "https://"+serverName+":8448", serverName, "", discover)
	}
	return s.dcrURL(ctx, serverName, "https://"+fromSRV, fromSRV, "", discover)
}

// getURLFromWK tries to get Federation API URL from /.well-known/matrix/server (step 3).
// Resolution follows https://spec.matrix.org/v1.18/server-server-api/#resolving-server-names
func (s *Server) getURLFromWK(ctx context.Context, serverName string, discover bool) (ssURL, ssHost, dialIP string) {
	log := apm.Log(ctx).With().Str("server", serverName).Logger()
	fromWellKnown, err := s.parseServerWellKnown(ctx, serverName)
	if err != nil {
		log.Warn().Err(err).Msg("failed to parse /.well-known/matrix/server")
		return "", "", ""
	}

	// Steps 3.1 / 3.2: delegated value has explicit port (covers "[ipv6]:port" and "host:port").
	if _, _, err := net.SplitHostPort(fromWellKnown); err == nil {
		return s.dcrURL(ctx, serverName, "https://"+fromWellKnown, fromWellKnown, "", discover)
	}

	// Step 3.1: bare IP literal with no port, default to 8448 without SRV lookup.
	if ip := net.ParseIP(fromWellKnown); ip != nil {
		host := fromWellKnown
		if ip.To4() == nil {
			host = "[" + fromWellKnown + "]"
		}
		return s.dcrURL(ctx, serverName, "https://"+host+":8448", fromWellKnown, "", discover)
	}

	// Steps 3.3 / 3.4 / 3.5: try discovering port via SRV (matrix-fed first, then legacy matrix)
	fromSRV, err := s.parseSRV(ctx, "matrix-fed", fromWellKnown)
	if err != nil {
		fromSRV, err = s.parseSRV(ctx, "matrix", fromWellKnown)
	}
	if err != nil {
		// if all SRV lookups fail, assume default port 8448
		return s.dcrURL(ctx, serverName, "https://"+fromWellKnown+":8448", fromWellKnown, "", discover)
	}

	fromSRVHost := strings.Split(fromSRV, ":")[0]
	// if SRV target matches well-known host, use SRV port as-is
	if fromSRVHost == fromWellKnown {
		return s.dcrURL(ctx, serverName, "https://"+fromSRV, fromWellKnown, "", discover)
	}
	// else, lookup A/AAAA for SRV target and pin the dial to that IP
	ips, err := net.DefaultResolver.LookupHost(ctx, fromSRVHost)
	if err != nil || len(ips) == 0 {
		return s.dcrURL(ctx, serverName, "https://"+fromWellKnown+":8448", fromWellKnown, "", discover)
	}
	_, port, err := net.SplitHostPort(fromSRV)
	if err != nil {
		log.Warn().Err(err).Str("srv", fromSRV).Msg("failed to parse SRV host:port, using default port 8448")
		port = "8448"
	}
	if net.ParseIP(ips[0]) == nil {
		log.Warn().Str("ip", ips[0]).Msg("resolved SRV target is not a valid IP")
		return s.dcrURL(ctx, serverName, "https://"+fromWellKnown+":8448", fromWellKnown, "", discover)
	}
	// SRV target differs from the delegated host: keep fromWellKnown in the URL (Host + SNI + cert), pin the dial to the resolved IP. dialContext brackets IPv6 via net.JoinHostPort.
	return s.dcrURL(ctx, serverName, "https://"+fromWellKnown+":"+port, fromWellKnown, ips[0], discover)
}

// lookupKeys requests /_matrix/key/v2/server by serverName
func (s *Server) lookupKeys(ctx context.Context, serverName string, discover bool) (*matrixKeyResp, error) {
	log := apm.Log(ctx).With().Str("server", serverName).Logger()

	ctx, serverURL, serverHost := s.getURL(ctx, serverName, discover)
	keysURL, err := url.Parse(serverURL + "/_matrix/key/v2/server")
	if err != nil {
		log.Warn().Err(err).Msg("failed to parse keys URL")
		return nil, err
	}
	resp, err := utils.Get(ctx, keysURL.String(), serverHost)
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

// notaryLookupKeys returns signed serverName's keys for notary use
func (s *Server) notaryLookupKeys(ctx context.Context, serverName string, validUntilTS int64) []byte {
	log := apm.Log(ctx).With().Str("server", serverName).Logger()
	ctx, cancel := context.WithTimeout(ctx, utils.DefaultTimeout)
	defer cancel()

	var keysResp *matrixKeyResp
	var err error
	cached, isCached := s.keysCache.Get(serverName)
	if isCached {
		keysResp = &cached
	}
	if keysResp == nil || keysResp.ValidUntilTS <= validUntilTS {
		isCached = false
		keysResp, err = s.lookupKeys(ctx, serverName, true)
	}
	if err != nil {
		log.Warn().Err(err).Msg("cannot lookup server keys")
		return nil
	}
	if keysResp == nil {
		log.Warn().Msg("no server keys found")
		return nil
	}
	if keysResp.ServerName != serverName {
		log.Warn().Str("discovered", keysResp.ServerName).Msg("server name mismatch in keys response")
		return nil
	}
	keyPayload, err := s.signJSON(keysResp)
	if err != nil {
		log.Error().Err(err).Msg("cannot sign key payload")
		return nil
	}

	if !isCached {
		// TODO: validate signatures
		s.keysCache.Add(serverName, *keysResp)
	}
	return keyPayload
}

// queryKeys returns serverName's keys
func (s *Server) queryKeys(ctx context.Context, serverName string) map[string]ed25519.PublicKey {
	cached, ok := s.keysCache.Get(serverName)
	if ok {
		keys := map[string]ed25519.PublicKey{}
		for id, data := range cached.VerifyKeys {
			pub, err := base64.RawStdEncoding.DecodeString(data["key"])
			if err != nil {
				continue
			}
			keys[id] = pub
		}
		return keys
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
	// TODO: validate signatures
	s.keysCache.Add(serverName, *resp)

	keys := map[string]ed25519.PublicKey{}
	for id, data := range resp.VerifyKeys {
		pub, err := base64.RawStdEncoding.DecodeString(data["key"])
		if err != nil {
			log.Warn().Err(err).Msg("failed to decode server key")
			continue
		}
		keys[id] = pub
	}
	return keys
}
