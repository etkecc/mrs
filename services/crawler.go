package services

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/pemistahl/lingua-go"
	"github.com/xxjwxc/gowp/workpool"
	"gitlab.com/etke.cc/go/msc1929"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

// RoomsBatch is maximum rooms parsed/stored at once
const RoomsBatch = 10000

type Crawler struct {
	servers     []string
	publicURL   string
	parsing     bool
	discovering bool
	eachrooming bool
	fed         FederationService
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

type FederationService interface {
	QueryPublicRooms(serverName, limit, since string) (*model.RoomDirectoryResponse, error)
	QueryServerName(serverName string) string
}

var (
	matrixMediaFallbacks = []string{"https://matrix-client.matrix.org"}
	matrixDialer         = &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 120 * time.Second,
	}
	matrixClient = &http.Client{
		Timeout: 120 * time.Second,
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

// NewCrawler service
func NewCrawler(servers []string, publicURL string, fedSvc FederationService, robots RobotsService, block BlocklistService, data DataRepository, detector lingua.LanguageDetector) *Crawler {
	return &Crawler{
		publicURL: publicURL,
		servers:   servers,
		robots:    robots,
		block:     block,
		fed:       fedSvc,
		data:      data,
		detector:  detector,
	}
}

// discoverServerNames performs basic sanitization and checks if server is online
func (m *Crawler) discoverServerNames(servers []string, workers int) []string {
	discovered := []string{}
	chunks := utils.Chunks(servers, 1000)
	log.Printf("discovering server names of %d servers using %d workers in %d chunks", len(servers), workers, len(chunks))
	for chunkID, chunk := range chunks {
		wp := workpool.New(workers)
		for _, server := range chunk {
			name := server
			wp.Do(func() error {
				uri, err := url.Parse("https://" + name)
				if err == nil {
					name = uri.Hostname()
				}
				name = m.fed.QueryServerName(name)
				if name != "" {
					discovered = append(discovered, name)
				}
				return nil
			})
		}
		wp.Wait() //nolint:errcheck
		log.Printf("[%d/%d] server names discovery in progress: %d of %d names discovered", chunkID, len(chunks), len(discovered), len(servers))
	}
	discovered = utils.Uniq(discovered)
	log.Printf("server names discovery finished - from %d servers intended for discovery, only %d are reachable", len(servers), len(discovered))
	return discovered
}

// DiscoverServers across federation and remove invalid ones
func (m *Crawler) DiscoverServers(workers int) error {
	if m.discovering {
		log.Println("servers discovery already in progress, ignoring request")
		return nil
	}
	m.discovering = true
	defer func() { m.discovering = false }()

	servers := utils.MergeSlices(utils.MapKeys(m.data.AllServers()), m.servers)
	discoveredServers := m.discoverServerNames(servers, workers)

	validServers := []string{}
	wp := workpool.New(workers)
	for _, server := range discoveredServers {
		name := server
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

func (m *Crawler) discoverServer(name string) (valid bool, err error) {
	if m.block.ByServer(name) {
		log.Println(name, "server is not eligible: blocked")
		return false, nil
	}

	if !m.robots.Allowed(name, RobotsTxtPublicRooms) {
		log.Println(name, "server is not eligible: robots.txt")
		m.block.Add(name)
		return false, nil
	}

	server := &model.MatrixServer{
		Name:      name,
		Online:    true,
		UpdatedAt: time.Now().UTC(),
	}

	if contacts := m.getServerContacts(name); contacts != nil {
		server.Contacts = *contacts
	}

	err = m.validateDiscoveredServer(name)
	if err != nil {
		log.Println(name, "server is not eligible:", err)
		m.block.Add(name)
		return false, nil
	}

	log.Println(name, "server is eligible")
	return true, m.data.AddServer(server)
}

// AddServers by name in bulk, intended for HTTP API
func (m *Crawler) AddServers(names []string, workers int) {
	wp := workpool.New(workers)
	discoveredServers := m.discoverServerNames(names, workers)
	validServers := []string{}
	for _, server := range discoveredServers {
		name := server
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
func (m *Crawler) AddServer(name string) int {
	name = m.fed.QueryServerName(name)
	if name == "" {
		return http.StatusUnprocessableEntity
	}

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
func (m *Crawler) AllServers() map[string]string {
	return m.data.AllServers()
}

// ParseRooms across all discovered servers
func (m *Crawler) ParseRooms(workers int) {
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
func (m *Crawler) EachRoom(handler func(roomID string, data *model.MatrixRoom)) {
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
func (m *Crawler) getMediaURLs(serverName, mediaID string) []string {
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

func (m *Crawler) downloadAvatar(serverName, mediaID string) (io.ReadCloser, string) {
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

func (m *Crawler) GetAvatar(serverName string, mediaID string) (io.Reader, string) {
	avatar, contentType := m.downloadAvatar(serverName, mediaID)
	converted, ok := utils.Avatar(avatar)
	if ok {
		contentType = utils.AvatarMIME
	}
	return converted, contentType
}

// validateDiscoveredServer performs simple check to public rooms endpoint
// to ensure discovered server is actually alive
func (m *Crawler) validateDiscoveredServer(name string) error {
	_, err := m.fed.QueryPublicRooms(name, "1", "")
	return err
}

// getServerContacts as per MSC1929
func (m *Crawler) getServerContacts(name string) *model.MatrixServerContacts {
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
func (m *Crawler) roomAllowed(name string, room *model.MatrixRoom) bool {
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
func (m *Crawler) getPublicRooms(name string) {
	var since string
	var added int
	limit := "10000"
	for {
		start := time.Now()
		resp, err := m.fed.QueryPublicRooms(name, limit, since)
		if err != nil {
			log.Println(name, "cannot query public rooms:", err)
			return
		}
		if len(resp.Chunk) == 0 {
			log.Println(name, "no public rooms available")
			return
		}

		added += len(resp.Chunk)
		for _, rdRoom := range resp.Chunk {
			room := rdRoom.Convert()
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
