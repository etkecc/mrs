package matrix

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/matrix-org/gomatrixserverlib"

	"github.com/etkecc/go-apm"
	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
	"github.com/etkecc/mrs/internal/version"
)

const (
	MatrixSearchLimit = 100 // default matrix (!) search limit
	devhost           = "localhost"
)

// Server server
type Server struct {
	cfg              configService
	keys             []*model.Key
	data             dataRepository
	media            mediaService
	search           searchService
	blocklist        blocklistService
	wellknownServer  []byte        // /.well-known/matrix/server contents
	wellknownClient  []byte        // /.well-known/matrix/client contents
	wellknownSupport []byte        // /.well-known/matrix/support contents
	versionServer    []byte        // /_matrix/federation/v1/version contents
	versionClient    []byte        // /_matrix/client/versions contents
	keyServer        matrixKeyResp // /_matrix/key/v2/server template
	discoverFunc     func(context.Context, string) int
	surlsCache       *lru.Cache[string, string]
	curlsCache       *lru.Cache[string, string]
	keysCache        *lru.Cache[string, map[string]ed25519.PublicKey]
	namesCache       *lru.Cache[string, string]
}

// NewServer creates new matrix server
func NewServer(cfg configService, data dataRepository, media mediaService, search searchService, blocklist blocklistService) (*Server, error) {
	keysCache, err := lru.New[string, map[string]ed25519.PublicKey](100000)
	if err != nil {
		return nil, err
	}
	namesCache, err := lru.New[string, string](100000)
	if err != nil {
		return nil, err
	}
	surlsCache, err := lru.New[string, string](100000)
	if err != nil {
		return nil, err
	}
	curlsCache, err := lru.New[string, string](100000)
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:        cfg,
		data:       data,
		media:      media,
		search:     search,
		blocklist:  blocklist,
		surlsCache: surlsCache,
		curlsCache: curlsCache,
		keysCache:  keysCache,
		namesCache: namesCache,
	}
	if err := s.initWellKnown(cfg.Get().Public.API); err != nil {
		return nil, err
	}
	if err := s.initKeys(cfg.Get().Matrix.Keys); err != nil {
		return nil, err
	}
	if err := s.initVersion(); err != nil {
		return nil, err
	}
	s.initKeyServer()

	return s, nil
}

// SetDiscover func
func (s *Server) SetDiscover(discover func(context.Context, string) int) {
	s.discoverFunc = discover
}

func (s *Server) MakeJoin() {
	ctx := apm.NewContext()
	apiURLStr := s.getURL(ctx, "etke.cc", false)
	apiURL, err := url.Parse(apiURLStr)
	if err != nil {
		panic(err)
	}
	apiURL = apiURL.JoinPath("/_matrix/federation/v1/make_join/!IyxAXBqViWHZfUkWjh:etke.cc/@test-make-join:matrixrooms.info")
	query := apiURL.Query()
	query.Add("ver", "10")
	query.Add("ver", "11")
	apiURL.RawQuery = query.Encode()

	path := "/" + apiURL.EscapedPath()
	if apiURL.RawQuery != "" {
		path += "?" + apiURL.RawQuery
	}
	fmt.Println("Making join request to", apiURL.String())
	authHeaders, err := s.Authorize("etke.cc", http.MethodGet, path, nil)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL.String(), http.NoBody)
	if err != nil {
		panic(err)
	}
	for _, h := range authHeaders {
		req.Header.Add("Authorization", h)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", version.UserAgent)
	resp, err := utils.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("%s\nResponse: %s\n", resp.Status, string(body))
	if resp.StatusCode != http.StatusOK {
		panic(fmt.Errorf("unexpected status code: %d", resp.StatusCode))
	}
	var makeJoinResp map[string]any
	if err := json.Unmarshal(body, &makeJoinResp); err != nil {
		panic(fmt.Errorf("cannot unmarshal join response: %w", err))
	}

	// SEND_JOIN
	evtJoin := makeJoinResp["event"].(map[string]any)
	evtJoin["origin"] = "matrixrooms.info"
	evtJoin["origin_server_ts"] = time.Now().UnixMilli()
	unsignedEvtJoin, _ := json.Marshal(evtJoin)
	evt, err := gomatrixserverlib.
		MustGetRoomVersion(gomatrixserverlib.RoomVersion(makeJoinResp["room_version"].(string))).
		NewEventFromUntrustedJSON(unsignedEvtJoin)
	if err != nil {
		panic(fmt.Errorf("cannot create event from untrusted JSON: %w", err))
	}
	evtID := evt.EventID()

	apiURL, err = url.Parse(apiURLStr)
	if err != nil {
		panic(err)
	}
	apiURL = apiURL.JoinPath(fmt.Sprintf(
		"/_matrix/federation/v2/send_join/%s/%s",
		"!IyxAXBqViWHZfUkWjh:etke.cc",
		evtID,
	))
	path = "/" + apiURL.EscapedPath()
	// TODO: something is wrong with this
	authHeaders, err = s.Authorize("etke.cc", http.MethodPut, path, evtJoin)
	if err != nil {
		panic(err)
	}
	// TODO: OR this
	signed, _ := s.signJSON(evtJoin)
	req, err = http.NewRequestWithContext(ctx, http.MethodPut, apiURL.String(), bytes.NewReader(signed))
	if err != nil {
		panic(err)
	}
	for _, h := range authHeaders {
		req.Header.Add("Authorization", h)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", version.UserAgent)
	dump, _ := httputil.DumpRequest(req, true)
	fmt.Printf("Request:\n%s\n", dump)
	resp, err = utils.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ = io.ReadAll(resp.Body)
	fmt.Printf("%s\nResponse: %s\n", resp.Status, string(body))
	if resp.StatusCode != http.StatusOK {
		panic(fmt.Errorf("unexpected status code: %d", resp.StatusCode))
	}
}
