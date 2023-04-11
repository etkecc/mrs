package services

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/xxjwxc/gowp/workpool"
	"gitlab.com/etke.cc/int/mrs/model"
)

type Matrix struct {
	servers     []string
	parsing     bool
	discovering bool
	eachrooming bool
	data        DataRepository
}

type DataRepository interface {
	AddServer(string, string) error
	GetServer(string) (string, error)
	RemoveServer(string) error
	EachServer(func(string, string))
	AddRoom(string, model.MatrixRoom) error
	GetRoom(string) (model.MatrixRoom, error)
	EachRoom(func(string, model.MatrixRoom))
}

var matrixClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 2*time.Second)
		},
	},
}

type matrixClientWellKnown struct {
	Homeserver matrixClientWellKnownHomeserver `json:"m.homeserver"`
}

type matrixClientWellKnownHomeserver struct {
	URL string `json:"base_url"`
}

type matrixRoomsResp struct {
	Chunk     []model.MatrixRoom `json:"chunk"`
	NextBatch string             `json:"next_batch"`
	Total     int                `json:"total_room_count_estimate"`
}

// NewMatrix service
func NewMatrix(servers []string, data DataRepository) *Matrix {
	return &Matrix{
		servers: servers,
		data:    data,
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
func (m *Matrix) DiscoverServers(workers int) {
	if m.discovering {
		log.Println("servers discovery already in progress, ignoring request")
		return
	}
	m.discovering = true
	defer func() { m.discovering = false }()

	wp := workpool.New(workers)
	for _, server := range m.servers {
		name := server
		wp.Do(func() error {
			serverURL, err := m.data.GetServer(name)
			if err != nil {
				return err
			}
			if serverURL != "" {
				return nil
			}

			serverURL = m.discoverServer(name)
			if serverURL == "" {
				return m.data.RemoveServer(name)
			}
			return m.data.AddServer(name, serverURL)
		})
	}
	wp.Wait() //nolint:errcheck
}

// ParseRooms across all discovered servers
func (m *Matrix) ParseRooms(workers int, indexfunc func(serverName string, roomID string, room model.MatrixRoom)) error {
	if m.parsing {
		log.Println("rooms parsing already in progress, ignoring request")
		return nil
	}
	m.parsing = true
	defer func() { m.parsing = false }()

	wp := workpool.New(workers)
	m.data.EachServer(func(srvName, srvUrl string) {
		name := srvName
		serverURL := srvUrl
		wp.Do(func() error {
			log.Println(name, "parsing rooms...")
			m.parseServerRooms(name, serverURL, indexfunc)
			return nil
		})
	})
	return wp.Wait()
}

// EachRoom allows to work with each known room
func (m *Matrix) EachRoom(handler func(roomID string, data model.MatrixRoom)) {
	if m.eachrooming {
		log.Println("iterating over each room is already in progress, ignoring request")
		return
	}
	m.eachrooming = true
	defer func() { m.eachrooming = false }()

	m.data.EachRoom(handler)
}

func (m *Matrix) parseServerRooms(name, serverURL string, indexfunc func(serverName string, roomID string, room model.MatrixRoom)) {
	ch := make(chan model.MatrixRoom)
	go m.getPublicRooms(name, serverURL, ch)
	for room := range ch {
		m.data.AddRoom(room.ID, room) //nolint:errcheck
		indexfunc(name, room.ID, room)
	}
}

// discoverServer resolves matrix server domain into actual server's url using well-known delegation
// inspired by https://github.com/mautrix/go/blob/master/client.go#L103
func (m *Matrix) discoverServer(name string) string {
	serverURL, _ := m.data.GetServer(name) //nolint:errcheck
	if serverURL != "" {
		return serverURL
	}

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

	m.data.AddServer(name, wellKnown.Homeserver.URL) //nolint:errcheck
	return wellKnown.Homeserver.URL
}

// getPublicRooms reads public rooms of the given server from the matrix client-server api
// and sends them into channel
func (m *Matrix) getPublicRooms(name, serverURL string, ch chan model.MatrixRoom) {
	var since string
	var added int
	for {
		resp := m.getPublicRoomsPage(name, serverURL, since)
		if resp == nil || len(resp.Chunk) == 0 {
			close(ch)
			break
		}
		added += len(resp.Chunk)

		start := time.Now()
		for _, room := range resp.Chunk {
			room.Server = name
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

func (m *Matrix) getPublicRoomsPage(name, serverURL, since string) *matrixRoomsResp {
	endpoint := serverURL + "/_matrix/client/v3/publicRooms?limit=500"
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
