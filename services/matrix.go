package services

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pemistahl/lingua-go"
	"github.com/xxjwxc/gowp/workpool"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

type Matrix struct {
	servers     []string
	publicURL   string
	parsing     bool
	discovering bool
	eachrooming bool
	data        DataRepository
	detector    lingua.LanguageDetector
}

type DataRepository interface {
	AddServer(*model.MatrixServer) error
	GetServer(string) (string, error)
	GetServerInfo(string) (*model.MatrixServer, error)
	AllServers() map[string]string
	AllOnlineServers() map[string]string
	RemoveServer(string) error
	AddRoom(string, *model.MatrixRoom) error
	GetRoom(string) (*model.MatrixRoom, error)
	EachRoom(func(string, *model.MatrixRoom))
	BanRoom(string) error
	UnbanRoom(string) error
}

var matrixMediaFallbacks = []string{"https://matrix-client.matrix.org"}

var matrixClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 10*time.Second)
		},
	},
}

type matrixClientWellKnown struct {
	Homeserver matrixClientWellKnownHomeserver `json:"m.homeserver"`
}

type matrixClientWellKnownHomeserver struct {
	URL string `json:"base_url"`
}

type matrixClientVersions struct {
	Versions []string `json:"versions"`
}

type matrixRoomsResp struct {
	Chunk     []*model.MatrixRoom `json:"chunk"`
	NextBatch string              `json:"next_batch"`
	Total     int                 `json:"total_room_count_estimate"`
}

// NewMatrix service
func NewMatrix(servers []string, publicURL string, data DataRepository, detector lingua.LanguageDetector) *Matrix {
	return &Matrix{
		publicURL: publicURL,
		servers:   servers,
		data:      data,
		detector:  detector,
	}
}

func matrixClientCall(endpoint string) (*http.Response, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Matrix Rooms Search")

	return matrixClient.Do(req)
}

// DiscoverServers across federation and reject invalid ones
//
//nolint:gocognit // TODO
func (m *Matrix) DiscoverServers(workers int) error {
	if m.discovering {
		log.Println("servers discovery already in progress, ignoring request")
		return nil
	}
	m.discovering = true
	defer func() { m.discovering = false }()

	dataservers := utils.MapKeys(m.data.AllServers())
	servers := utils.MergeSlices(dataservers, m.servers)
	wp := workpool.New(workers)
	for _, server := range servers {
		name := server
		wp.Do(func() error {
			server := &model.MatrixServer{
				Name:      name,
				Online:    true,
				UpdatedAt: time.Now().UTC(),
			}
			server.URL = m.discoverServerWellKnown(name)
			if server.URL != "" && m.validateDiscoveredServer(name, server.URL) {
				return m.data.AddServer(server)
			}
			server.URL = m.discoverServerDirect(name)
			if server.URL != "" && m.validateDiscoveredServer(name, server.URL) {
				return m.data.AddServer(server)
			}

			server.Online = false
			return m.data.AddServer(server)
		})
	}
	return wp.Wait() //nolint:errcheck
}

// AddServers by name in bulk, intended for HTTP API
//
//nolint:gocognit // TODO
func (m *Matrix) AddServers(names []string, workers int) {
	wp := workpool.New(workers)
	for _, server := range names {
		name := server
		wp.Do(func() error {
			log.Println(name, "discovering...")
			existingURL, _ := m.data.GetServer(name) //nolint:errcheck
			if existingURL != "" {
				return nil
			}

			server := &model.MatrixServer{
				Name:      name,
				Online:    true,
				UpdatedAt: time.Now().UTC(),
			}

			server.URL = m.discoverServerWellKnown(name)
			if server.URL == "" {
				server.URL = m.discoverServerDirect(name)
			}
			if server.URL == "" {
				log.Println(name, "server not found")
				server.Online = false
				return m.data.AddServer(server)
			}
			if !m.validateDiscoveredServer(name, server.URL) {
				log.Println(name, "server is not eligible")
				server.Online = false
				return m.data.AddServer(server)
			}

			err := m.data.AddServer(server)
			if err != nil {
				log.Println(name, "cannot add server", err)
				return nil
			}

			return nil
		})
	}
	wp.Wait() //nolint:errcheck
}

// AddServer by name, intended for HTTP API
// returns http status code to send to the reporter
func (m *Matrix) AddServer(name string) int {
	existingURL, _ := m.data.GetServer(name) //nolint:errcheck
	if existingURL != "" {
		return http.StatusAlreadyReported
	}

	server := &model.MatrixServer{
		Name:      name,
		Online:    true,
		UpdatedAt: time.Now().UTC(),
	}

	server.URL = m.discoverServerWellKnown(name)
	if server.URL == "" {
		server.URL = m.discoverServerDirect(name)
	}
	if server.URL == "" {
		return http.StatusUnprocessableEntity
	}
	if !m.validateDiscoveredServer(name, server.URL) {
		return http.StatusUnprocessableEntity
	}

	err := m.data.AddServer(server)
	if err != nil {
		log.Println(name, "cannot add server", err)
		return http.StatusInternalServerError
	}

	return http.StatusCreated
}

// AllServers returns map of all known servers
func (m *Matrix) AllServers() map[string]string {
	return m.data.AllServers()
}

// ParseRooms across all discovered servers
func (m *Matrix) ParseRooms(workers int) error {
	if m.parsing {
		log.Println("rooms parsing already in progress, ignoring request")
		return nil
	}
	m.parsing = true
	defer func() { m.parsing = false }()

	servers := m.data.AllOnlineServers()
	wp := workpool.New(workers)
	for srvName, srvURL := range servers {
		name := srvName
		serverURL := srvURL
		wp.Do(func() error {
			log.Println(name, "parsing rooms...")
			m.parseServerRooms(name, serverURL)
			return nil
		})
	}
	return wp.Wait()
}

// EachRoom allows to work with each known room
func (m *Matrix) EachRoom(handler func(roomID string, data *model.MatrixRoom)) {
	if m.eachrooming {
		log.Println("iterating over each room is already in progress, ignoring request")
		return
	}
	m.eachrooming = true
	defer func() { m.eachrooming = false }()

	m.data.EachRoom(handler)
}

// getMediaServers returns list of HTTP urls of the same media ID.
// that list contains the requested server plus fallback media servers
func (m *Matrix) getMediaURLs(serverName, mediaID string) []string {
	urls := make([]string, 0, len(matrixMediaFallbacks)+1)
	for _, serverURL := range matrixMediaFallbacks {
		urls = append(urls, serverURL+"/_matrix/media/v3/download/"+serverName+"/"+mediaID)
	}
	serverURL, err := m.data.GetServer(serverName)
	if err != nil && serverURL != "" {
		urls = append(urls, serverURL+"/_matrix/media/v3/download/"+serverName+"/"+mediaID)
	}

	return urls
}

func (m *Matrix) GetAvatar(serverName string, mediaID string) (io.ReadCloser, string) {
	datachan := make(chan map[string]io.ReadCloser, 1)
	for _, avatarURL := range m.getMediaURLs(serverName, mediaID) {
		go func(datachan chan map[string]io.ReadCloser, avatarURL string) {
			select {
			case <-datachan:
				return
			default:
				resp, err := http.Get(avatarURL)
				if err != nil {
					return
				}
				if resp.StatusCode != http.StatusOK {
					return
				}
				datachan <- map[string]io.ReadCloser{
					resp.Header.Get("Content-Type"): resp.Body,
				}
			}
		}(datachan, avatarURL)
	}

	for contentType, avatar := range <-datachan {
		close(datachan)
		return avatar, contentType
	}

	return nil, ""
}

func (m *Matrix) parseServerRooms(name, serverURL string) {
	ch := make(chan *model.MatrixRoom)
	go m.getPublicRooms(name, serverURL, ch)
	for room := range ch {
		m.data.AddRoom(room.ID, room) //nolint:errcheck
	}
}

// discoverServerWellKnown resolves matrix server domain into actual server's url using well-known delegation
// inspired by https://github.com/mautrix/go/blob/master/client.go#L103
func (m *Matrix) discoverServerWellKnown(name string) string {
	resp, err := matrixClientCall("https://" + name + "/.well-known/matrix/client")
	if err != nil {
		log.Println(name, "cannot get matrix delegation information", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Println(name, "matrix delegation does not exists")
		return ""
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(name, "cannot read matrix delegation information", err)
		return ""
	}

	var wellKnown matrixClientWellKnown
	err = json.Unmarshal(data, &wellKnown)
	if err != nil {
		log.Println(name, "cannot unmarshal matrix delegation information", err)
		return ""
	}
	if wellKnown.Homeserver.URL == "" {
		log.Println(name, "matrix delegation information is empty")
		return ""
	}

	if strings.HasPrefix(wellKnown.Homeserver.URL, "https://") {
		return wellKnown.Homeserver.URL
	}
	return "https://" + wellKnown.Homeserver.URL
}

// discoverServerDirect tries to discover matrix server directly by supported protocol versions endpoint
func (m *Matrix) discoverServerDirect(name string) string {
	resp, err := matrixClientCall("https://" + name + "/_matrix/client/versions")
	if err != nil {
		log.Println(name, "cannot get matrix client versions", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Println(name, "matrix client versions not found")
		return ""
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(name, "cannot read matrix client versions", err)
		return ""
	}

	var clientVersions matrixClientVersions
	err = json.Unmarshal(data, &clientVersions)
	if err != nil {
		log.Println(name, "cannot unmarshal matrix client versions", err)
		return ""
	}
	if len(clientVersions.Versions) == 0 {
		log.Println(name, "matrix client versions are empty")
		return ""
	}

	return "https://" + name
}

// validateDiscoveredServer performs simple check to public rooms endpoint
// to ensure discovered server is actually alive
func (m *Matrix) validateDiscoveredServer(name, serverURL string) bool {
	return m.getPublicRoomsPage(name, serverURL, "1", "") != nil
}

// roomPreviewAvailable checks if room can be previewed on view.matrix.org
func (m *Matrix) roomPreviewAvailable(roomID string) bool {
	endpoint := model.MatrixPreviewURL.JoinPath("/room", roomID, "/").String()
	resp, err := matrixClientCall(endpoint)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// getPublicRooms reads public rooms of the given server from the matrix client-server api
// and sends them into channel
func (m *Matrix) getPublicRooms(name, serverURL string, ch chan *model.MatrixRoom) {
	var since string
	var added int
	limit := "10000"
	for {
		resp := m.getPublicRoomsPage(name, serverURL, limit, since)
		if resp == nil || len(resp.Chunk) == 0 {
			close(ch)
			break
		}
		added += len(resp.Chunk)

		start := time.Now()
		for _, room := range resp.Chunk {
			room.Parse(m.detector, m.publicURL, m.roomPreviewAvailable(room.ID))
			ch <- room
		}
		log.Println(name, "added", len(resp.Chunk), "rooms (", added, "of", resp.Total, ") took", time.Since(start))

		if resp.NextBatch == "" {
			close(ch)
			break
		}

		since = resp.NextBatch
	}
}

func (m *Matrix) getPublicRoomsPage(name, serverURL, limit, since string) *matrixRoomsResp {
	endpoint := serverURL + "/_matrix/client/v3/publicRooms?limit=" + limit
	if since != "" {
		endpoint = endpoint + "&since=" + url.QueryEscape(since)
	}

	resp, err := matrixClientCall(endpoint)
	if err != nil {
		log.Println(name, "cannot get public rooms", err)
		return nil
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(name, "cannot read public rooms", err)
		return nil
	}

	var roomsResp *matrixRoomsResp
	err = json.Unmarshal(data, &roomsResp)
	if err != nil {
		log.Println(name, "cannot unmarshal public rooms", err)
		return nil
	}
	return roomsResp
}
