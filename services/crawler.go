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
	cfg         ConfigService
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
	Slice() []string
	Reset()
}

type RobotsService interface {
	Allowed(serverName, endpoint string) bool
}

type DataRepository interface {
	AddServer(*model.MatrixServer) error
	GetServer(string) (string, error)
	GetServerInfo(string) (*model.MatrixServer, error)
	EachServerInfo(func(string, *model.MatrixServer))
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
	QueryVersion(serverName string) (string, string, error)
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
func NewCrawler(cfg ConfigService, fedSvc FederationService, robots RobotsService, block BlocklistService, data DataRepository, detector lingua.LanguageDetector) *Crawler {
	return &Crawler{
		cfg:      cfg,
		robots:   robots,
		block:    block,
		fed:      fedSvc,
		data:     data,
		detector: detector,
	}
}

func (m *Crawler) validateServer(name string) (string, bool) {
	uri, err := url.Parse("https://" + name)
	if err == nil {
		name = uri.Hostname()
	}

	// check if online
	name = m.fed.QueryServerName(name)
	if name == "" {
		return "", false
	}

	// check if not blocked
	if m.block.ByServer(name) {
		return "", false
	}

	// check if federateable
	if _, _, ferr := m.fed.QueryVersion(name); ferr != nil {
		return "", false
	}

	return name, true
}

// validateServers performs basic sanitization, checks if server is online and federateable
func (m *Crawler) validateServers(servers []string, workers int) []string {
	discovered := []string{}
	chunks := utils.Chunks(servers, 1000)
	log.Printf("validating %d servers using %d workers in %d chunks", len(servers), workers, len(chunks))
	for i, chunk := range chunks {
		wp := workpool.New(workers)
		for _, server := range chunk {
			srvName := server
			wp.Do(func() error {
				name, ok := m.validateServer(srvName)
				if ok {
					discovered = append(discovered, name)
				}
				return nil
			})
		}
		wp.Wait() //nolint:errcheck
		log.Printf("[%d/%d] servers validation in progress: %d of %d servers are valid", i+1, len(chunks), len(discovered), len(servers))
	}
	discovered = utils.Uniq(discovered)
	log.Printf("servers validation finished - from %d servers intended for discovery, only %d are valid (online and federateable)", len(servers), len(discovered))
	return discovered
}

func (m *Crawler) loadServers() []string {
	servers := utils.MergeSlices(utils.MapKeys(m.data.AllServers()), m.cfg.Get().Servers)
	log.Printf("loaded %d servers from config and db", len(servers))
	fromRooms := []string{}
	m.EachRoom(func(_ string, data *model.MatrixRoom) {
		fromRooms = append(fromRooms, data.Server)
		if data.Alias != "" {
			fromRooms = append(fromRooms, utils.ServerFrom(data.Alias))
		}
	})
	fromRooms = utils.Uniq(fromRooms)
	servers = utils.MergeSlices(servers, fromRooms)
	log.Printf("loaded %d servers from already parsed rooms, total known servers = %d", len(fromRooms), len(servers))
	return servers
}

// DiscoverServers across federation and remove invalid ones
func (m *Crawler) DiscoverServers(workers int) error {
	if m.discovering {
		log.Println("servers discovery already in progress, ignoring request")
		return nil
	}
	m.discovering = true
	defer func() { m.discovering = false }()

	discoveredServers := m.validateServers(m.loadServers(), workers)

	// remove blocked servers
	m.block.Reset()
	toRemove := utils.RemoveFromSlice(discoveredServers, m.block.Slice())
	m.data.RemoveServers(toRemove)

	wp := workpool.New(workers)
	for _, server := range discoveredServers {
		name := server
		wp.Do(func() error {
			_, err := m.discoverServer(name)
			return err
		})
	}
	return wp.Wait()
}

func (m *Crawler) discoverServer(name string) (valid bool, err error) {
	server := &model.MatrixServer{
		Name:      name,
		Online:    true,
		UpdatedAt: time.Now().UTC(),
	}

	if !m.robots.Allowed(name, RobotsTxtPublicRooms) {
		log.Println(name, "server is not eligible: robots.txt")
		return false, m.data.AddServer(server) // not indexable yet, but online
	}

	if contacts := m.getServerContacts(name); contacts != nil {
		server.Contacts = *contacts
	}

	if _, err = m.fed.QueryPublicRooms(name, "1", ""); err != nil {
		log.Println(name, "server is not eligible:", err)
		return false, m.data.AddServer(server) // not indexable yet, but online
	}

	log.Println(name, "server is eligible")
	server.Indexable = true
	return true, m.data.AddServer(server)
}

// AddServers by name in bulk, intended for HTTP API
func (m *Crawler) AddServers(names []string, workers int) {
	wp := workpool.New(workers)
	discoveredServers := m.validateServers(names, workers)
	validServers := []string{}
	for _, server := range discoveredServers {
		srvName := server
		wp.Do(func() error {
			existingURL, _ := m.data.GetServer(srvName) //nolint:errcheck
			if existingURL != "" {
				return nil
			}
			name, ok := m.validateServer(srvName)
			if !ok {
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
}

// AddServer by name, intended for HTTP API
// returns http status code to send to the reporter
func (m *Crawler) AddServer(name string) int {
	existingURL, _ := m.data.GetServer(name) //nolint:errcheck
	if existingURL != "" {
		return http.StatusAlreadyReported
	}

	name, ok := m.validateServer(name)
	if !ok {
		return http.StatusUnprocessableEntity
	}

	valid, err := m.discoverServer(name)
	if err != nil {
		log.Println(name, "cannot add server", err)
	}
	if !valid {
		return http.StatusUnprocessableEntity
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

			room.Parse(m.detector, m.cfg.Get().Public.API)
			m.data.AddRoomBatch(room)
		}
		log.Println(name, "added", len(resp.Chunk), "rooms (", added, "of", resp.Total, ") took", time.Since(start))

		if resp.NextBatch == "" {
			return
		}

		since = resp.NextBatch
	}
}
