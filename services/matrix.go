package services

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"gitlab.com/etke.cc/int/mrs/model"
)

type Matrix struct {
	servers []string
	data    DataRepository
}

type DataRepository interface {
	AddServer(string, string) error
	GetServer(string) (string, error)
	AddRoom(string, model.Entry) error
	GetRoom(string) (model.Entry, error)
}

var matrixClient = &http.Client{
	Timeout: 30 * time.Second,
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
	PrevBatch string             `json:"prev_batch"`
}

// NewMatrix service
func NewMatrix(servers []string, data DataRepository) Matrix {
	return Matrix{servers, data}
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

func (m *Matrix) GetRooms(server string) []model.Entry {
	serverURL := m.discoverServer(server)
	rooms := m.getPublicRooms(serverURL)
	for _, room := range rooms {
		m.data.AddRoom(room.ID, room) //nolint:errcheck
	}

	return rooms
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
		log.Println(name, "cannot get matrix delegation information")
		return name
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Println(name, "matrix delegation does not exists")
		return name
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(name, "cannot read matrix delegation information")
		return name
	}

	var wellKnown matrixClientWellKnown
	err = json.Unmarshal(data, &wellKnown)
	if err != nil {
		log.Println("cannot unmarshal matrix delegation information")
		return name
	}

	m.data.AddServer(name, wellKnown.Homeserver.URL) //nolint:errcheck
	return wellKnown.Homeserver.URL
}

// getPublicRooms reads public rooms of the given server from the matrix client-server api
func (m *Matrix) getPublicRooms(serverURL string) []model.Entry {
	resp, err := matrixClientCall(serverURL + "/_matrix/client/v3/publicRooms")
	if err != nil {
		log.Println(serverURL, "cannot get public rooms")
		return nil
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(serverURL, "cannot read public rooms")
		return nil
	}

	var roomsResp matrixRoomsResp
	err = json.Unmarshal(data, &roomsResp)
	if err != nil {
		log.Println(serverURL, "cannot unmarshal public rooms")
		return nil
	}
	entries := make([]model.Entry, 0, len(roomsResp.Chunk))
	for _, room := range roomsResp.Chunk {
		entries = append(entries, model.Entry(room))
	}

	return entries
}
