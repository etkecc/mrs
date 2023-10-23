package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pemistahl/lingua-go"
	"github.com/xxjwxc/gowp/workpool"
	"gitlab.com/etke.cc/go/msc1929"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
	"gitlab.com/etke.cc/mrs/api/version"
)

// RoomsBatch is maximum rooms parsed/stored at once
const RoomsBatch = 10000

type Matrix struct {
	servers     []string
	proxyURL    string
	proxyToken  string
	publicURL   string
	parsing     bool
	discovering bool
	eachrooming bool
	block       BlocklistService
	robots      RobotsService
	data        DataRepository
	detector    lingua.LanguageDetector
}

type BlocklistService interface {
	Add(server string)
	ByID(matrixID string) bool
	ByServer(server string) bool
}

type RobotsService interface {
	Allowed(serverName, endpoint string) bool
}

type DataRepository interface {
	AddServer(*model.MatrixServer) error
	GetServer(string) (string, error)
	GetServerInfo(string) (*model.MatrixServer, error)
	AllServers() map[string]string
	RemoveServer(string) error
	RemoveServers([]string)
	AddRoomBatch(*model.MatrixRoom)
	FlushRoomBatch()
	GetRoom(string) (*model.MatrixRoom, error)
	EachRoom(func(string, *model.MatrixRoom))
	GetBannedRooms(...string) ([]string, error)
	RemoveRooms([]string)
	BanRoom(string) error
	UnbanRoom(string) error
	GetReportedRooms(...string) (map[string]string, error)
	ReportRoom(string, string) error
	UnreportRoom(string) error
	IsReported(string) bool
}

var (
	matrixMediaFallbacks = []string{"https://matrix-client.matrix.org"}
	matrixDialer         = &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 90 * time.Second,
	}
	matrixClient = &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxConnsPerHost:     100,
			MaxIdleConnsPerHost: 5,
			TLSHandshakeTimeout: 10 * time.Second,
			DialContext:         matrixDialer.DialContext,
			Dial:                matrixDialer.Dial,
		},
	}
)

type matrixRoomsResp struct {
	Chunk     []*model.MatrixRoom `json:"chunk"`
	NextBatch string              `json:"next_batch"`
	Total     int                 `json:"total_room_count_estimate"`
}

// NewMatrix service
func NewMatrix(servers []string, proxyURL, proxyToken, publicURL string, robots RobotsService, block BlocklistService, data DataRepository, detector lingua.LanguageDetector) *Matrix {
	return &Matrix{
		proxyURL:   proxyURL,
		proxyToken: proxyToken,
		publicURL:  publicURL,
		servers:    servers,
		robots:     robots,
		block:      block,
		data:       data,
		detector:   detector,
	}
}

func (m *Matrix) call(ctx context.Context, endpoint string, withAuth bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	if withAuth {
		req.Header.Set("Connection", "Keep-Alive")
		req.Header.Set("Authorization", "Bearer "+m.proxyToken)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", version.UserAgent)

	return matrixClient.Do(req)
}

// DiscoverServers across federation and remove invalid ones
func (m *Matrix) DiscoverServers(workers int) error {
	if m.discovering {
		log.Println("servers discovery already in progress, ignoring request")
		return nil
	}
	m.discovering = true
	defer func() { m.discovering = false }()

	servers := utils.MergeSlices(utils.MapKeys(m.data.AllServers()), m.servers)
	validServers := []string{}
	wp := workpool.New(workers)
	for _, server := range servers {
		name := server
		name = m.sanitizeServerName(name)
		wp.Do(func() error {
			valid, err := m.discoverServer(name)
			if valid {
				validServers = append(validServers, name)
			}
			return err
		})
	}
	perr := wp.Wait()
	toRemove := utils.RemoveFromSlice(servers, validServers)
	log.Printf("removing %d invalid/offline/blocked servers", len(toRemove))
	m.data.RemoveServers(toRemove)

	return perr
}

// sanitizeServerName returns sanitized server name
func (m *Matrix) sanitizeServerName(name string) string {
	uri, err := url.Parse("https://" + name)
	if err != nil {
		log.Println(name, "cannot parse url", err)
		return name
	}
	return uri.Hostname()
}

func (m *Matrix) discoverServer(name string) (valid bool, err error) {
	if m.block.ByServer(name) {
		return false, nil
	}

	if !m.robots.Allowed(name, RobotsTxtPublicRooms) {
		log.Println(name, "robots.txt disallow parsing")
		m.block.Add(name)
		return false, nil
	}

	log.Println(name, "discovering...")
	server := &model.MatrixServer{
		Name:      name,
		Online:    true,
		UpdatedAt: time.Now().UTC(),
	}

	if contacts := m.getServerContacts(name); contacts != nil {
		server.Contacts = *contacts
	}

	if !m.validateDiscoveredServer(name) {
		log.Println(name, "server is not eligible, removing")
		m.block.Add(name)
		return false, nil
	}

	return true, m.data.AddServer(server)
}

// AddServers by name in bulk, intended for HTTP API
func (m *Matrix) AddServers(names []string, workers int) {
	wp := workpool.New(workers)
	validServers := []string{}
	for _, server := range names {
		name := server
		name = m.sanitizeServerName(name)
		wp.Do(func() error {
			existingURL, _ := m.data.GetServer(name) //nolint:errcheck
			if existingURL != "" {
				return nil
			}

			valid, err := m.discoverServer(name)
			if valid {
				validServers = append(validServers, name)
			}
			return err
		})
	}

	wp.Wait() //nolint:errcheck
	toRemove := utils.RemoveFromSlice(names, validServers)
	m.data.RemoveServers(toRemove)
}

// AddServer by name, intended for HTTP API
// returns http status code to send to the reporter
func (m *Matrix) AddServer(name string) int {
	name = m.sanitizeServerName(name)
	existingURL, _ := m.data.GetServer(name) //nolint:errcheck
	if existingURL != "" {
		return http.StatusAlreadyReported
	}

	valid, err := m.discoverServer(name)
	if !valid {
		m.data.RemoveServer(name) //nolint:errcheck
	}

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
func (m *Matrix) ParseRooms(workers int) {
	if m.parsing {
		log.Println("rooms parsing already in progress, ignoring request")
		return
	}
	m.parsing = true
	defer func() { m.parsing = false }()

	servers := utils.MapKeys(m.data.AllServers())

	total := len(servers)
	if total < workers {
		workers = total
	}
	wp := workpool.New(workers)
	log.Println("parsing rooms of", total, "servers using", workers, "workers")
	for _, srvName := range servers {
		name := srvName
		if m.block.ByServer(name) {
			if err := m.data.RemoveServer(name); err != nil {
				log.Println(name, "cannot remove blocked server", err)
			}
			continue
		}

		wp.Do(func() error {
			log.Println(name, "parsing rooms...")
			m.getPublicRooms(name)
			return nil
		})
	}
	wp.Wait() //nolint:errcheck
	m.data.FlushRoomBatch()
}

// EachRoom allows to work with each known room
func (m *Matrix) EachRoom(handler func(roomID string, data *model.MatrixRoom)) {
	if m.eachrooming {
		log.Println("iterating over each room is already in progress, ignoring request")
		return
	}
	m.eachrooming = true
	defer func() { m.eachrooming = false }()

	toRemove := []string{}
	m.data.EachRoom(func(id string, room *model.MatrixRoom) {
		if !m.roomAllowed(room.Server, room) {
			toRemove = append(toRemove, id)
			return
		}

		handler(id, room)
	})
	m.data.RemoveRooms(toRemove)
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

func (m *Matrix) downloadAvatar(serverName, mediaID string) (io.ReadCloser, string) {
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

func (m *Matrix) GetAvatar(serverName string, mediaID string) (io.Reader, string) {
	avatar, contentType := m.downloadAvatar(serverName, mediaID)
	converted, ok := utils.Avatar(avatar)
	if ok {
		contentType = utils.AvatarMIME
	}
	return converted, contentType
}

// validateDiscoveredServer performs simple check to public rooms endpoint
// to ensure discovered server is actually alive
func (m *Matrix) validateDiscoveredServer(name string) bool {
	return m.getPublicRoomsPage(name, "1", "") != nil
}

// getServerContacts as per MSC1929
func (m *Matrix) getServerContacts(name string) *model.MatrixServerContacts {
	resp, err := msc1929.Get(name)
	if err != nil {
		log.Println(name, "cannot get server contacts", err)
		return nil
	}
	if resp.IsEmpty() {
		return nil
	}
	return &model.MatrixServerContacts{
		Emails: utils.Uniq(resp.Emails()),
		MXIDs:  utils.Uniq(resp.MatrixIDs()),
		URL:    resp.SupportPage,
	}
}

// roomAllowed check if room is allowed (by blocklist and robots.txt)
func (m *Matrix) roomAllowed(name string, room *model.MatrixRoom) bool {
	if room.ID == "" {
		return false
	}
	if m.block.ByID(room.ID) {
		return false
	}
	if m.block.ByID(room.Alias) {
		return false
	}
	if m.block.ByServer(room.Server) {
		return false
	}
	if m.block.ByServer(name) {
		return false
	}

	return m.robots.Allowed(name, fmt.Sprintf(RobotsTxtPublicRoom, room.ID))
}

// getPublicRooms reads public rooms of the given server from the matrix client-server api
// and sends them into channel
func (m *Matrix) getPublicRooms(name string) {
	var since string
	var added int
	limit := "10000"
	for {
		start := time.Now()
		resp := m.getPublicRoomsPage(name, limit, since)
		if resp == nil || len(resp.Chunk) == 0 {
			log.Println(name, "response is empty")
			return
		}

		added += len(resp.Chunk)
		for _, room := range resp.Chunk {
			if !m.roomAllowed(name, room) {
				added--
				continue
			}

			room.Parse(m.detector, m.publicURL)
			m.data.AddRoomBatch(room)
		}
		log.Println(name, "added", len(resp.Chunk), "rooms (", added, "of", resp.Total, ") took", time.Since(start))

		if resp.NextBatch == "" {
			return
		}

		since = resp.NextBatch
	}
}

func (m *Matrix) getPublicRoomsPage(name, limit, since string) *matrixRoomsResp {
	endpoint := m.proxyURL + "/_matrix/client/v3/publicRooms?server=" + name + "&limit=" + limit
	if since != "" {
		endpoint = endpoint + "&since=" + url.QueryEscape(since)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := m.call(ctx, endpoint, true)
	if err != nil {
		if strings.Contains(err.Error(), "Timeout") || strings.Contains(err.Error(), "deadline") {
			log.Println(name, "cannot get public rooms (timeout)")
			return nil
		}

		log.Println(name, "cannot get public rooms", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println(name, "cannot get public rooms", resp.Status)
		return nil
	}

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
