package services

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/matrix-org/gomatrixserverlib"
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/etke.cc/mrs/api/config"
	"gitlab.com/etke.cc/mrs/api/metrics"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
	"gitlab.com/etke.cc/mrs/api/version"
)

const devhost = "localhost"

// matrixKeyResp is response of /_matrix/key/v2/server
type matrixKeyResp struct {
	ServerName    string                       `json:"server_name"`
	ValidUntilTS  int64                        `json:"valid_until_ts"`
	VerifyKeys    map[string]map[string]string `json:"verify_keys"`
	OldVerifyKeys map[string]map[string]any    `json:"old_verify_keys"`
	Signatures    map[string]map[string]string `json:"signatures,omitempty"`
}

type wellKnownServerResp struct {
	Host string `json:"m.server"`
}

type versionResp struct {
	Server map[string]string `json:"server"`
}

type matrixAuth struct {
	Origin      string
	Destination string
	KeyID       string
	Signature   []byte
}

type matrixSearchService interface {
	Search(query, sortBy string, limit, offset int) ([]*model.Entry, int, error)
}

// Matrix server
type Matrix struct {
	name         string
	keys         []*model.Key
	search       matrixSearchService
	wellknown    []byte        // /.well-known/matrix/server contents
	version      []byte        // /_matrix/federation/v1/version contents
	keyServer    matrixKeyResp // /_matrix/key/v2/server template
	discoverFunc func(string) int
	urlsCache    *lru.Cache[string, string]
	keysCache    *lru.Cache[string, map[string]ed25519.PublicKey]
	namesCache   *lru.Cache[string, string]
}

// NewMatrix creates new matrix server
func NewMatrix(cfg *config.Config, search matrixSearchService) (*Matrix, error) {
	keysCache, err := lru.New[string, map[string]ed25519.PublicKey](100000)
	if err != nil {
		return nil, err
	}
	namesCache, err := lru.New[string, string](100000)
	if err != nil {
		return nil, err
	}
	urlsCache, err := lru.New[string, string](100000)
	if err != nil {
		return nil, err
	}

	m := &Matrix{
		name:       cfg.Matrix.ServerName,
		search:     search,
		urlsCache:  urlsCache,
		keysCache:  keysCache,
		namesCache: namesCache,
	}
	if err := m.initWellKnown(cfg.Public.API); err != nil {
		return nil, err
	}
	if err := m.initKeys(cfg.Matrix.Keys); err != nil {
		return nil, err
	}
	if err := m.initVersion(); err != nil {
		return nil, err
	}
	m.initKeyServer()

	return m, nil
}

// SetDiscover func
func (m *Matrix) SetDiscover(discover func(string) int) {
	m.discoverFunc = discover
}

// GetWellKnown returns json-eligible response for /.well-known/matrix/server
func (m *Matrix) GetWellKnown() []byte {
	return m.wellknown
}

// GetVersion returns json-eligible response for /_matrix/federation/v1/version
func (m *Matrix) GetVersion() []byte {
	return m.version
}

// GetKeyServer returns jsonblob-eligible response for /_matrix/key/v2/server
func (m *Matrix) GetKeyServer() []byte {
	resp := m.keyServer
	resp.ValidUntilTS = time.Now().UTC().Add(24 * time.Hour).UnixMilli()
	payload, err := m.signJSON(resp)
	if err != nil {
		log.Println("ERROR: cannot sign payload:", err)
	}
	return payload
}

// PublicRooms returns /_matrix/federation/v1/publicRooms response
func (m *Matrix) PublicRooms(req *http.Request, rdReq *model.RoomDirectoryRequest) (int, []byte) {
	origin, err := m.ValidateAuth(req)
	log.Printf("auth: origin=%s err=%v", origin, err)
	if err != nil {
		return http.StatusUnauthorized, nil
	}

	defer metrics.SearchQueries.With(prometheus.Labels{
		"api":    "matrix",
		"server": origin,
	}).Inc()

	limit := rdReq.Limit
	offset := utils.StringToInt(rdReq.Since)
	entries, total, err := m.search.Search(rdReq.Filter.GenericSearchTerm, "", limit, offset)
	if err != nil {
		log.Println("ERROR: cannot search from matrix:", err)
		return http.StatusInternalServerError, nil
	}
	chunk := make([]*model.RoomDirectoryRoom, 0, len(entries))
	for _, entry := range entries {
		chunk = append(chunk, entry.RoomDirectory())
	}
	var prev int
	if offset >= limit {
		prev = offset - limit
	}

	value, err := utils.JSON(model.RoomDirectoryResponse{
		Chunk:     chunk,
		PrevBatch: strconv.Itoa(prev),
		NextBatch: strconv.Itoa(offset + len(chunk)),
		Total:     total,
	})
	if err != nil {
		log.Println("ERROR: cannot marshal room directory json:", err)
		return http.StatusInternalServerError, nil
	}
	return http.StatusOK, value
}

// ValidateAuth validates matrix auth
func (m *Matrix) ValidateAuth(r *http.Request) (serverName string, err error) {
	defer r.Body.Close()
	if m.name == devhost {
		log.Println("ignoring auth validation on dev host")
		return "ignored", nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	content := make(map[string]any)
	if len(body) > 0 {
		if jerr := json.Unmarshal(body, &content); jerr != nil {
			return "", jerr
		}
	}

	auths := m.parseAuths(r)
	if len(auths) == 0 {
		return "", fmt.Errorf("no auth provided")
	}
	obj := map[string]any{
		"method":      r.Method,
		"uri":         r.RequestURI,
		"origin":      auths[0].Origin,
		"destination": auths[0].Destination,
	}
	if len(content) > 0 {
		obj["content"] = content
	}
	canonical, err := utils.JSON(obj)
	if err != nil {
		return "", err
	}
	keys := m.queryKeys(auths[0].Origin)
	if len(keys) == 0 {
		return "", fmt.Errorf("no server keys available")
	}
	for _, auth := range auths {
		if err := m.validateAuth(obj, canonical, auth, keys); err != nil {
			log.Println("canonical", string(canonical))
			log.Println("content", content)
			log.Println("obj", obj)
			return "", err
		}
	}

	return auths[0].Origin, nil
}

// QueryServerName finds server name on the /_matrix/key/v2/server page
func (m *Matrix) QueryServerName(serverName string) string {
	cached, ok := m.namesCache.Get(serverName)
	if ok {
		return cached
	}
	discovered := ""
	resp, err := m.lookupKeys(serverName, false)
	if err == nil && resp != nil {
		discovered = resp.ServerName
	}
	m.namesCache.Add(serverName, discovered)
	return discovered
}

// QueryVersion from /_matrix/federation/v1/version
func (m *Matrix) QueryVersion(serverName string) (server, version string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.getURL(serverName, false)+"/_matrix/federation/v1/version", nil)
	if err != nil {
		return "", "", err
	}
	resp, err := matrixClient.Do(req)
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
	var vResp *versionResp
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
func (m *Matrix) QueryPublicRooms(serverName, limit, since string) (*model.RoomDirectoryResponse, error) {
	serverName = m.QueryServerName(serverName)
	if serverName == "" {
		return nil, fmt.Errorf("server is offline")
	}

	req, err := m.buildPublicRoomsReq(serverName, limit, since)
	if err != nil {
		return nil, err
	}

	resp, err := matrixClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // intended
		return nil, fmt.Errorf("cannot get public rooms: %s; %s", resp.Status, string(body))
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

// Authorize request
func (m *Matrix) Authorize(serverName, method, uri string, body any) ([]string, error) {
	obj := map[string]any{
		"method":      method,
		"uri":         uri,
		"origin":      m.name,
		"destination": serverName,
	}
	if body != nil {
		obj["content"] = body
	}
	signed, err := m.signJSON(obj)
	if err != nil {
		return nil, err
	}
	var objSigned map[string]any
	if jerr := json.Unmarshal(signed, &objSigned); jerr != nil {
		return nil, jerr
	}
	if _, ok := objSigned["signatures"]; !ok {
		return nil, fmt.Errorf("no signatures")
	}
	allSignatures, ok := objSigned["signatures"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cannot parse signatures: %v", objSigned["signatures"])
	}

	signatures, ok := allSignatures[m.name].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cannot parse own signatures: %v", allSignatures[m.name])
	}
	headers := make([]string, 0, len(signatures))
	for keyID, sig := range signatures {
		headers = append(headers, fmt.Sprintf(
			`X-Matrix origin="%s",destination="%s",key="%s",sig="%s"`,
			m.name, serverName, keyID, sig,
		))
	}
	return headers, nil
}

func (m *Matrix) buildPublicRoomsReq(serverName, limit, since string) (*http.Request, error) {
	apiURLStr := m.getURL(serverName, false)
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
	authHeaders, err := m.Authorize(serverName, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, apiURL.String(), nil)
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

func (m *Matrix) initKeys(strs []string) error {
	if len(strs) == 0 {
		return nil
	}
	keys := []*model.Key{}
	for _, str := range strs {
		key, err := model.KeyFrom(str)
		if err != nil {
			return err
		}
		keys = append(keys, key)
	}
	m.keys = keys
	return nil
}

func (m *Matrix) initWellKnown(apiURL string) error {
	uri, err := url.Parse(apiURL)
	if err != nil {
		return err
	}
	port := uri.Port()
	if port == "" {
		port = "443"
	}

	value, err := utils.JSON(map[string]string{
		"m.server": uri.Hostname() + ":" + port,
	})
	m.wellknown = value
	return err
}

func (m *Matrix) initVersion() error {
	value, err := utils.JSON(map[string]map[string]string{
		"server": {
			"name":    version.Name,
			"version": version.Version,
		},
	})
	m.version = value
	return err
}

func (m *Matrix) initKeyServer() {
	resp := matrixKeyResp{
		ServerName:    m.name,
		ValidUntilTS:  time.Now().UTC().Add(24 * time.Hour).UnixMilli(),
		VerifyKeys:    map[string]map[string]string{},
		OldVerifyKeys: map[string]map[string]any{},
	}
	for _, key := range m.keys {
		resp.VerifyKeys[key.ID] = map[string]string{"key": key.Public}
	}
	m.keyServer = resp
}

// signJSON using server keys
func (m *Matrix) signJSON(input any) ([]byte, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	for _, key := range m.keys {
		payload, err = gomatrixserverlib.SignJSON(m.name, gomatrixserverlib.KeyID(key.ID), key.Private, payload)
		if err != nil {
			return nil, err
		}
	}
	return payload, nil
}

func (m *Matrix) parseAuth(authorization string) *matrixAuth {
	auth := &matrixAuth{}
	paramsStr := strings.ReplaceAll(authorization, "X-Matrix ", "")
	paramsSlice := strings.Split(paramsStr, ",")
	for _, param := range paramsSlice {
		parts := strings.Split(param, "=")
		if len(parts) < 2 {
			continue
		}
		value := strings.Trim(parts[1], `"`)
		switch parts[0] {
		case "origin":
			auth.Origin = value
		case "destination":
			auth.Destination = value
		case "key":
			auth.KeyID = value
		case "sig":
			sig, err := base64.RawStdEncoding.DecodeString(value)
			if err != nil {
				log.Println("ERROR: cannot decode signature:", err)
				return nil
			}
			auth.Signature = sig
		}
	}
	if auth.Origin == "" || auth.KeyID == "" || len(auth.Signature) == 0 {
		return nil
	}
	return auth
}

func (m *Matrix) validateAuth(obj map[string]any, canonical []byte, auth *matrixAuth, keys map[string]ed25519.PublicKey) error {
	if auth.Origin != obj["origin"] {
		return fmt.Errorf("auth is from multiple servers")
	}
	if auth.Destination != obj["destination"] {
		return fmt.Errorf("auth is for multiple servers")
	}
	if auth.Destination != "" && auth.Destination != m.name {
		return fmt.Errorf("unknown destination")
	}

	key, ok := keys[auth.KeyID]
	if !ok {
		return fmt.Errorf("unknown key '%s'", auth.KeyID)
	}
	if !ed25519.Verify(key, canonical, auth.Signature) {
		return fmt.Errorf("failed signatures on '%s'", auth.KeyID)
	}

	return nil
}

// parseAuths parses Authorization headers,
// copied from https://github.com/turt2live/matrix-media-repo/blob/4da32e5739a8924e0cfcdde2daf4af4a90c2ff85/util/http.go#L52
func (m *Matrix) parseAuths(r *http.Request) []*matrixAuth {
	headers := r.Header.Values("Authorization")
	auths := make([]*matrixAuth, 0)
	for _, h := range headers {
		if !strings.HasPrefix(h, "X-Matrix ") {
			continue
		}
		auth := m.parseAuth(h)
		if auth != nil {
			auths = append(auths, auth)
		}
	}

	return auths
}

// parseWellKnown returns Federation API host:port
func (m *Matrix) parseWellKnown(serverName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+serverName+"/.well-known/matrix/server", nil)
	if err != nil {
		return "", err
	}
	resp, err := matrixClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("no /.well-known/matrix/server")
	}

	datab, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var wellknown *wellKnownServerResp
	if wkerr := json.Unmarshal(datab, &wellknown); wkerr != nil {
		return "", wkerr
	}
	if wellknown.Host == "" {
		return "", fmt.Errorf("/.well-known/matrix/server is empty")
	}
	parts := strings.Split(wellknown.Host, ":")
	if len(parts) == 0 {
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
func (m *Matrix) parseSRV(service, serverName string) (string, error) {
	_, addrs, err := net.LookupSRV(service, "tcp", serverName)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("no " + service + " SRV records")
	}
	return strings.Trim(addrs[0].Target, ".") + ":" + strconv.Itoa(int(addrs[0].Port)), nil
}

// dcrURL stands for discover-cache-and-return URL, shortcut for m.getURL
func (m *Matrix) dcrURL(serverName, url string, discover bool) string {
	m.urlsCache.Add(serverName, url)

	if m.discoverFunc != nil && discover {
		go m.discoverFunc(serverName)
	}

	return url
}

// getURL returns Federation API URL
func (m *Matrix) getURL(serverName string, discover bool) string {
	cached, ok := m.urlsCache.Get(serverName)
	if ok {
		return cached
	}

	fromWellKnown, err := m.parseWellKnown(serverName)
	if err == nil {
		return m.dcrURL(serverName, "https://"+fromWellKnown, discover)
	}
	fromSRV, err := m.parseSRV("matrix-fed", serverName)
	if err == nil {
		return m.dcrURL(serverName, "https://"+fromSRV, discover)
	}
	fromSRV, err = m.parseSRV("matrix", serverName)
	if err == nil {
		return m.dcrURL(serverName, "https://"+fromSRV, discover)
	}

	return m.dcrURL(serverName, "https://"+serverName, discover)
}

// lookupKeys requests /_matrix/key/v2/server by serverName
func (m *Matrix) lookupKeys(serverName string, discover bool) (*matrixKeyResp, error) {
	keysURL, err := url.Parse(m.getURL(serverName, discover) + "/_matrix/key/v2/server")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, keysURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", version.UserAgent)
	resp, err := matrixClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, err
	}
	datab, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var keysResp *matrixKeyResp
	if err := json.Unmarshal(datab, &keysResp); err != nil {
		return nil, err
	}
	return keysResp, nil
}

// queryKeys returns serverName's keys
func (m *Matrix) queryKeys(serverName string) map[string]ed25519.PublicKey {
	cached, ok := m.keysCache.Get(serverName)
	if ok {
		return cached
	}
	resp, err := m.lookupKeys(serverName, true)
	if err != nil {
		log.Println("ERROR: keys query failed:", err)
		return nil
	}
	if resp.ServerName != serverName {
		log.Println("ERROR: server name doesn't match")
		return nil
	}
	if resp.ValidUntilTS <= time.Now().UnixMilli() {
		log.Println("ERROR: server keys are expired")
	}
	keys := map[string]ed25519.PublicKey{}
	for id, data := range resp.VerifyKeys {
		pub, err := base64.RawStdEncoding.DecodeString(data["key"])
		if err != nil {
			log.Println("ERROR: failed to decode server key:", err)
			continue
		}
		keys[id] = pub
	}
	// TODO: verify signatures
	m.keysCache.Add(serverName, keys)
	return keys
}
