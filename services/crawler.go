package services

import (
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/pemistahl/lingua-go"
	"github.com/rs/zerolog"
	"github.com/xxjwxc/gowp/workpool"
	"gitlab.com/etke.cc/go/msc1929"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

type Crawler struct {
	v           ValidatorService
	cfg         ConfigService
	parsing     bool
	discovering bool
	eachrooming bool
	fed         FederationService
	block       BlocklistService
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
	HasServer(string) bool
	GetServer(string) (string, error)
	GetServerInfo(string) (*model.MatrixServer, error)
	EachServerInfo(func(string, *model.MatrixServer))
	AllServers() map[string]string
	BatchServers([]string) error
	RemoveServer(string) error
	RemoveServers([]string)
	AddRoomBatch(*model.MatrixRoom)
	FlushRoomBatch()
	GetRoom(string) (*model.MatrixRoom, error)
	EachRoom(func(string, *model.MatrixRoom))
	SetBiggestRooms([]string) error
	GetBannedRooms(...string) ([]string, error)
	RemoveRooms([]string)
	BanRoom(string) error
	UnbanRoom(string) error
	GetReportedRooms(...string) (map[string]string, error)
	ReportRoom(string, string) error
	UnreportRoom(string) error
	IsReported(string) bool
}

type ValidatorService interface {
	Domain(server string) bool
	IsOnline(server string) (string, bool)
	IsIndexable(server string) bool
	IsRoomAllowed(server string, room *model.MatrixRoom) bool
}

type FederationService interface {
	QueryPublicRooms(serverName, limit, since string) (*model.RoomDirectoryResponse, error)
	QueryServerName(serverName string) (string, error)
	QueryVersion(serverName string) (string, string, error)
	QueryCSURL(serverName string) string
}

var (
	matrixMediaFallbacks = []string{"https://matrix-client.matrix.org"}
	matrixClient         = &http.Client{Timeout: 120 * time.Second}
)

// NewCrawler service
func NewCrawler(cfg ConfigService, fedSvc FederationService, v ValidatorService, block BlocklistService, data DataRepository, detector lingua.LanguageDetector) *Crawler {
	return &Crawler{
		v:        v,
		cfg:      cfg,
		fed:      fedSvc,
		block:    block,
		data:     data,
		detector: detector,
	}
}

// validateServers performs basic sanitization, checks if server is online and federateable
func (m *Crawler) validateServers(servers *utils.List[string, string], workers int) *utils.List[string, string] {
	log := utils.Logger
	wp := workpool.New(workers)
	discovered := utils.NewList[string, string]()
	log.Info().Int("servers", servers.Len()).Int("workers", workers).Msg("validating servers")

	for _, server := range servers.Slice() {
		srvName := server
		wp.Do(func() error {
			name, ok := m.v.IsOnline(srvName)
			if ok {
				discovered.Add(name)
			}
			return nil
		})
	}
	go func(wp *workpool.WorkPool, log *zerolog.Logger) {
		for {
			if wp.IsDone() {
				return
			}

			log.Info().
				Int("valid", discovered.Len()).
				Int("of", servers.Len()).
				Msg("servers validation in progress")

			time.Sleep(1 * time.Minute)
		}
	}(wp, log)
	wp.Wait() //nolint:errcheck

	log.Info().Int("valid", discovered.Len()).Int("of", servers.Len()).Msg("servers validation finished")
	return discovered
}

func (m *Crawler) loadServers() *utils.List[string, string] {
	servers := utils.NewList[string, string]()
	servers.AddMapKeys(m.data.AllServers())
	servers.AddSlice(m.cfg.Get().Servers)
	utils.Logger.Info().Int("servers", servers.Len()).Msg("loaded servers from config and db")

	return servers
}

// DiscoverServers across federation and remove invalid ones
func (m *Crawler) DiscoverServers(workers int) error {
	if m.discovering {
		utils.Logger.Info().Msg("servers discovery already in progress, ignoring request")
		return nil
	}
	m.discovering = true
	defer func() { m.discovering = false }()

	allServers := m.loadServers()
	validServers := m.validateServers(allServers, workers)

	wp := workpool.New(workers)
	for _, server := range validServers.Slice() {
		name := server
		wp.Do(func() error {
			_, err := m.discoverServer(name)
			return err
		})
	}

	go func(wp *workpool.WorkPool) {
		for {
			if wp.IsDone() {
				return
			}

			utils.Logger.Info().
				Int("of", validServers.Len()).
				Msg("servers discovery in progress")

			time.Sleep(1 * time.Minute)
		}
	}(wp)
	err := wp.Wait()

	allServers.RemoveSlice(validServers.Slice())
	utils.Logger.Info().Int("offline", allServers.Len()).Msg("removing offline servers")
	m.data.RemoveServers(allServers.Slice())

	return err
}

func (m *Crawler) discoverServer(name string) (valid bool, err error) {
	server := &model.MatrixServer{
		Name:      name,
		URL:       m.fed.QueryCSURL(name),
		Online:    true,
		UpdatedAt: time.Now().UTC(),
	}

	if m.v.IsIndexable(name) {
		server.Indexable = true
	}

	if contacts := m.getServerContacts(name); contacts != nil {
		server.Contacts = *contacts
	}

	return true, m.data.AddServer(server)
}

// AddServers by name in bulk, intended for HTTP API
func (m *Crawler) AddServers(names []string, workers int) {
	wp := workpool.New(workers)
	list := utils.NewList[string, string]()
	list.AddSlice(names)
	discoveredServers := m.validateServers(list, workers)
	validServers := []string{}
	for _, server := range discoveredServers.Slice() {
		srvName := server
		wp.Do(func() error {
			if m.data.HasServer(srvName) {
				return nil
			}
			name, ok := m.v.IsOnline(srvName)
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
	if m.data.HasServer(name) {
		return http.StatusAlreadyReported
	}

	name, ok := m.v.IsOnline(name)
	if !ok {
		return http.StatusUnprocessableEntity
	}

	valid, err := m.discoverServer(name)
	if err != nil {
		utils.Logger.Warn().Err(err).Str("server", name).Msg("cannot add server")
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
		utils.Logger.Info().Msg("room parsing already in progress, ignoring request")
		return
	}
	m.parsing = true
	defer func() { m.parsing = false }()

	servers := utils.NewList[string, string]()
	servers.AddMapKeys(m.data.AllServers())
	servers.RemoveSlice(m.block.Slice())
	total := servers.Len()

	if total < workers {
		workers = total
	}
	wp := workpool.New(workers)
	utils.Logger.Info().Int("servers", total).Int("workers", workers).Msg("parsing rooms")
	for _, srvName := range servers.Slice() {
		name := srvName
		if m.block.ByServer(name) {
			servers.Remove(name)
			if err := m.data.RemoveServer(name); err != nil {
				utils.Logger.Error().Err(err).Str("server", name).Msg("cannot remove blocked server")
			}
			continue
		}
		if !m.v.Domain(name) {
			servers.Remove(name)
			if err := m.data.RemoveServer(name); err != nil {
				utils.Logger.Error().Err(err).Str("server", name).Msg("cannot remove server with invalid domain")
			}
			continue
		}

		wp.Do(func() error {
			m.getPublicRooms(servers, name)
			return nil
		})
	}

	go func(wp *workpool.WorkPool) {
		for {
			if wp.IsDone() {
				return
			}

			utils.Logger.Info().
				Int("of", servers.Len()).
				Msg("parsing rooms in progress")

			time.Sleep(1 * time.Minute)
		}
	}(wp)
	wp.Wait() //nolint:errcheck
	m.data.FlushRoomBatch()

	servers.RemoveSlice(m.block.Slice())
	if err := m.data.BatchServers(servers.Slice()); err != nil {
		utils.Logger.Error().Err(err).Msg("writing batch of servers failed")
	}

	m.calculateBiggestRooms()
}

func (m *Crawler) calculateBiggestRooms() {
	type roomCount struct {
		id      string
		members int
	}

	utils.Logger.Info().Msg("calculating biggest rooms...")
	started := time.Now().UTC()
	counts := []roomCount{}
	m.data.EachRoom(func(_ string, data *model.MatrixRoom) {
		counts = append(counts, roomCount{data.ID, data.Members})
	})

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].members > counts[j].members
	})
	ids := make([]string, 0, len(counts))
	for _, count := range counts {
		ids = append(ids, count.id)
	}
	utils.Logger.Info().Str("took", time.Since(started).String()).Msg("biggest rooms have been calculated, storing")
	if err := m.data.SetBiggestRooms(ids); err != nil {
		utils.Logger.Error().Err(err).Msg("cannot set biggest rooms")
	}
	utils.Logger.Info().Str("took", time.Since(started).String()).Msg("biggest rooms have been calculated and stored")
}

// EachRoom allows to work with each known room
func (m *Crawler) EachRoom(handler func(roomID string, data *model.MatrixRoom)) {
	if m.eachrooming {
		utils.Logger.Info().Msg("iterating over each room is already in progress, ignoring request")
		return
	}
	m.eachrooming = true
	defer func() { m.eachrooming = false }()

	toRemove := []string{}
	m.data.EachRoom(func(id string, room *model.MatrixRoom) {
		if !m.v.IsRoomAllowed(room.Server, room) {
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

// getPublicRooms reads public rooms of the given server from the matrix client-server api
// and sends them into channel
func (m *Crawler) getPublicRooms(servers *utils.List[string, string], name string) {
	var since string
	var added int
	limit := "10000"
	for {
		start := time.Now()
		resp, err := m.fed.QueryPublicRooms(name, limit, since)
		if err != nil {
			utils.Logger.Warn().Err(err).Str("server", name).Msg("cannot query public rooms")
			return
		}
		if len(resp.Chunk) == 0 {
			utils.Logger.Info().Str("server", name).Msg("no public rooms available")
			return
		}

		added += len(resp.Chunk)
		for _, rdRoom := range resp.Chunk {
			room := rdRoom.Convert()
			if !m.v.IsRoomAllowed(name, room) {
				added--
				continue
			}

			room.Parse(m.detector, m.cfg.Get().Public.API, m.cfg.Get().Matrix.ServerName)
			servers.AddSlice(room.Servers(m.cfg.Get().Matrix.ServerName))

			m.data.AddRoomBatch(room)
		}
		utils.Logger.
			Info().
			Str("server", name).
			Int("added", len(resp.Chunk)).
			Int("of", resp.Total).
			Str("took", time.Since(start).String()).
			Msg("added rooms")

		if resp.NextBatch == "" {
			return
		}

		since = resp.NextBatch
	}
}
