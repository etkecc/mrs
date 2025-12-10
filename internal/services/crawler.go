package services

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/etkecc/go-apm"
	"github.com/etkecc/go-kit"
	"github.com/etkecc/go-kit/workpool"
	"github.com/etkecc/go-msc1929"
	"github.com/pemistahl/lingua-go"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

type Crawler struct {
	v           ValidatorService
	cfg         ConfigService
	parsing     bool
	discovering bool
	eachrooming bool
	fed         FederationService
	block       BlocklistService
	media       MediaService
	data        DataRepository
	detector    lingua.LanguageDetector
}

type BlocklistService interface {
	Add(server string)
	ByID(matrixID string) bool
	ByServer(server string) bool
	Reset()
}

type DataRepository interface {
	AddServer(context.Context, *model.MatrixServer) error
	HasServer(context.Context, string) bool
	GetServerInfo(context.Context, string) (*model.MatrixServer, error)
	FilterServers(context.Context, func(server *model.MatrixServer) bool) map[string]*model.MatrixServer
	BatchServers(context.Context, []string) error
	MarkServersOffline(context.Context, []string)
	RemoveServer(context.Context, string) error
	RemoveServers(context.Context, []string)
	AddRoomBatch(context.Context, *model.MatrixRoom)
	FlushRoomBatch(context.Context)
	GetRoom(context.Context, string) (*model.MatrixRoom, error)
	AddRoomMapping(context.Context, string, string) error
	GetRoomMapping(context.Context, string) string
	RemoveRoomMapping(context.Context, string, string)
	RecreateRoomMapping(context.Context, map[string]string) error
	EachRoom(context.Context, func(string, *model.MatrixRoom) bool)
	SetBiggestRooms(context.Context, []string) error
	GetBannedRooms(context.Context, ...string) ([]string, error)
	RemoveRooms(context.Context, []string)
	BanRoom(context.Context, string) error
	UnbanRoom(context.Context, string) error
	GetReportedRooms(context.Context, ...string) (map[string]string, error)
	ReportRoom(context.Context, string, string, string) error
	UnreportRoom(context.Context, string) error
	UnreportAll(context.Context) error
	IsReported(context.Context, string) bool
}

type ValidatorService interface {
	Domain(server string) bool
	IsOnline(ctx context.Context, server string) (serverName, serverSoftware, serverVersion string, isOnline bool)
	IsIndexable(ctx context.Context, server string) bool
	IsRoomAllowed(server string, room *model.MatrixRoom) bool
}

type FederationService interface {
	QueryPublicRooms(ctx context.Context, serverName, limit, since string) (*model.RoomDirectoryResponse, error)
	QueryDirectoryExternal(ctx context.Context, roomAlias string) (*model.QueryDirectoryResponse, error)
	QueryServerName(ctx context.Context, serverName string) (string, error)
	QueryVersion(ctx context.Context, serverName string) (string, string, error)
	QueryCSURL(ctx context.Context, serverName string) string
}

// NewCrawler service
func NewCrawler(cfg ConfigService, fedSvc FederationService, v ValidatorService, block BlocklistService, media MediaService, data DataRepository, detector lingua.LanguageDetector) *Crawler {
	return &Crawler{
		v:        v,
		cfg:      cfg,
		fed:      fedSvc,
		block:    block,
		media:    media,
		data:     data,
		detector: detector,
	}
}

// DiscoverServers across federation and remove invalid ones
func (m *Crawler) DiscoverServers(ctx context.Context, workers int, overrideList ...*kit.List[string, string]) {
	log := apm.Log(ctx)
	if m.discovering {
		log.Info().Msg("servers discovery already in progress, ignoring request")
		return
	}
	m.discovering = true
	defer func() { m.discovering = false }()

	var servers *kit.List[string, string]
	if len(overrideList) > 0 {
		servers = overrideList[0]
	} else {
		servers = m.loadServers(ctx)
	}

	offline := m.discoverServers(ctx, servers, workers)

	log.Info().Int("offline", offline.Len()).Msg("marking offline servers")
	m.data.MarkServersOffline(ctx, offline.Slice())
}

// AddServers by name in bulk, intended for HTTP API
func (m *Crawler) AddServers(ctx context.Context, names []string, workers int) {
	servers := kit.NewListFrom(names)
	// exclude already known servers first
	for _, server := range servers.Slice() {
		if m.data.HasServer(ctx, server) {
			servers.Remove(server)
		}
	}

	m.discoverServers(ctx, servers, workers)
}

// AddServer by name, intended for HTTP API
// returns http status code to send to the reporter
func (m *Crawler) AddServer(ctx context.Context, name string) int {
	if m.data.HasServer(ctx, name) {
		return http.StatusAlreadyReported
	}

	server := m.discoverServer(ctx, name)
	if server == nil {
		return http.StatusUnprocessableEntity
	}

	return http.StatusCreated
}

// ParseRooms across all discovered servers
func (m *Crawler) ParseRooms(ctx context.Context, workers int) {
	log := apm.Log(ctx)
	if m.parsing {
		log.Info().Msg("room parsing already in progress, ignoring request")
		return
	}

	m.parsing = true
	defer func() { m.parsing = false }()

	servers := kit.NewList[string, string]()
	indexable := m.IndexableServers(ctx)
	for _, server := range indexable {
		if m.block.ByServer(server) {
			continue
		}
		servers.Add(server)
	}
	slice := servers.Slice()
	total := len(slice)

	if total < workers {
		workers = total
	}
	wp := workpool.New(workers)
	discoveredServers := kit.NewList[string, string]()
	log.Info().Int("servers", total).Int("workers", workers).Msg("parsing rooms")
	for _, srvName := range slice {
		name := srvName
		wp.Do(func() {
			serversFromRooms := m.getPublicRooms(ctx, name)
			discoveredServers.AddSlice(serversFromRooms.Slice())
		})
	}

	wp.Run()
	m.data.FlushRoomBatch(ctx)
	discoveredServers.RemoveSlice(servers.Slice())
	log.
		Info().
		Int("of", servers.Len()).
		Int("discovered_servers", discoveredServers.Len()).
		Msg("parsing rooms has been finished")

	m.DiscoverServers(ctx, m.cfg.Get().Workers.Discovery, discoveredServers)

	m.afterRoomParsing(ctx)
}

// EachRoom allows to work with each known room
func (m *Crawler) EachRoom(ctx context.Context, handler func(roomID string, data *model.MatrixRoom) bool) {
	log := apm.Log(ctx)
	if m.eachrooming {
		log.Info().Msg("iterating over each room is already in progress, ignoring request")
		return
	}
	m.eachrooming = true
	defer func() { m.eachrooming = false }()

	toRemove := []string{}
	m.data.EachRoom(ctx, func(id string, room *model.MatrixRoom) bool {
		if !m.v.IsRoomAllowed("", room) {
			toRemove = append(toRemove, id)
			return false
		}

		return handler(id, room)
	})
	m.data.RemoveRooms(ctx, toRemove)
}

// OnlineServers returns all known online servers
func (m *Crawler) OnlineServers(ctx context.Context) []string {
	return kit.MapKeys(m.OnlineServersObjects(ctx))
}

// OnlineServersObjects returns all online servers
func (m *Crawler) OnlineServersObjects(ctx context.Context) map[string]*model.MatrixServer {
	return m.data.FilterServers(ctx, func(server *model.MatrixServer) bool {
		return server.Online
	})
}

// IndexableServers returns all known indexable servers
func (m *Crawler) IndexableServers(ctx context.Context) []string {
	return kit.MapKeys(m.data.FilterServers(ctx, func(server *model.MatrixServer) bool {
		return server.Online && server.Indexable
	}))
}

func (m *Crawler) GetRoom(ctx context.Context, roomIDorAlias string) (*model.MatrixRoom, error) {
	roomID := roomIDorAlias
	if utils.IsValidAlias(roomIDorAlias) {
		if mapped := m.data.GetRoomMapping(ctx, roomIDorAlias); mapped != "" {
			roomID = mapped
		}
	}

	room, err := m.data.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, nil
	}
	if !m.v.IsRoomAllowed(room.Server, room) {
		return nil, nil
	}
	return room, nil
}

func (m *Crawler) loadServers(ctx context.Context) *kit.List[string, string] {
	log := apm.Log(ctx)
	log.Info().Msg("loading servers")
	servers := kit.NewList[string, string]()
	servers.AddSlice(m.cfg.Get().Servers)
	log.Info().Int("servers", servers.Len()).Msg("loaded servers from config")
	servers.AddSlice(kit.MapKeys(m.data.FilterServers(ctx, func(_ *model.MatrixServer) bool {
		return true
	})))
	log.Info().Int("servers", servers.Len()).Msg("loaded servers from config and db")

	return servers
}

// discoverServer parses server information
func (m *Crawler) discoverServer(ctx context.Context, rawName string) *model.MatrixServer {
	if m.block.ByServer(rawName) {
		apm.Log(ctx).Info().Str("server", rawName).Msg("server is blocked, skipping")
		return &model.MatrixServer{Name: rawName, Online: false}
	}
	name, software, version, ok := m.v.IsOnline(ctx, rawName)
	if name == "" {
		return &model.MatrixServer{Name: rawName, Online: false}
	}

	server := &model.MatrixServer{
		Name:     name,
		Software: software,
		Version:  version,
		URL:      m.fed.QueryCSURL(ctx, name),
		Contacts: m.getServerContacts(ctx, name),
		Online:   ok,
		OnlineAt: time.Now().UTC(),
	}

	if m.v.IsIndexable(ctx, name) {
		server.Indexable = true
	}

	if err := m.data.AddServer(ctx, server); err != nil {
		apm.Log(ctx).
			Error().Err(err).Msg("cannot store server")
	}
	return server
}

// discoverServers parses servers information and returns lists of OFFLINE servers
func (m *Crawler) discoverServers(ctx context.Context, servers *kit.List[string, string], workers int) (offline *kit.List[string, string]) {
	wp := workpool.New(workers)
	log := apm.Log(ctx)
	online := kit.NewList[string, string]()
	offline = kit.NewList[string, string]()
	indexable := kit.NewList[string, string]() // just for stats
	log.Info().Int("servers", servers.Len()).Int("workers", workers).Msg("validating servers")

	for _, server := range servers.Slice() {
		srvName := server
		wp.Do(func() {
			server := m.discoverServer(ctx, srvName)
			serverName := srvName
			if server.Name != "" {
				serverName = server.Name
			}

			if server.Online {
				online.Add(serverName)
			} else {
				offline.Add(serverName)
			}
			if server.Indexable {
				indexable.Add(serverName)
			}
		})
	}
	wp.Run()

	log.Info().
		Int("online", online.Len()).
		Int("offline", offline.Len()).
		Int("indexable", indexable.Len()).
		Int("of", servers.Len()).
		Msg("servers discovery finished")
	return offline
}

func (m *Crawler) removeOldOfflineServers(ctx context.Context) {
	log := apm.Log(ctx)
	threshold := time.Now().UTC().AddDate(0, -1, 0)
	servers := m.data.FilterServers(ctx, func(server *model.MatrixServer) bool {
		return !server.Online && server.OnlineAt.Before(threshold)
	})
	if len(servers) == 0 {
		log.Info().Msg("no old offline servers to remove")
		return
	}

	toRemove := kit.MapKeys(servers)
	log.Info().Int("servers", len(toRemove)).Msg("removing old offline servers")
	m.data.RemoveServers(ctx, toRemove)
}

//nolint:gocognit // TODO: refactor
func (m *Crawler) afterRoomParsing(ctx context.Context) {
	type roomCount struct {
		id      string
		members int
	}

	log := apm.Log(ctx)
	log.Info().Msg("after room parsing......")
	started := time.Now().UTC()
	counts := []roomCount{}
	toRemove := map[string]string{}
	mapping := map[string]string{}
	m.data.EachRoom(ctx, func(id string, data *model.MatrixRoom) bool {
		if started.Sub(data.ParsedAt) >= 24*7*time.Hour { // parsed more than a week ago
			toRemove[id] = data.Avatar
			return false
		}
		counts = append(counts, roomCount{data.ID, data.Members})
		mapping[data.ID] = data.Alias
		return false
	})

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].members > counts[j].members
	})
	ids := make([]string, 0, len(counts))
	for _, count := range counts {
		ids = append(ids, count.id)
	}
	log.Info().Str("took", time.Since(started).String()).Msg("biggest rooms have been calculated, storing")
	if err := m.data.SetBiggestRooms(ctx, ids); err != nil {
		log.Error().Err(err).Msg("cannot set biggest rooms")
	}
	log.Info().Str("took", time.Since(started).String()).Msg("biggest rooms have been calculated and stored")

	if len(toRemove) > 0 {
		log.Info().Int("rooms", len(toRemove)).Msg("removing rooms last updated more than a week ago...")
		toRemoveSlice := kit.MapKeys(toRemove)
		for _, mxcURL := range toRemove {
			if mxcURL == "" {
				continue
			}
			parts := strings.Split(strings.TrimPrefix(mxcURL, "mxc://"), "/")
			if len(parts) != 2 {
				continue
			}
			m.media.Delete(ctx, parts[0], parts[1])
		}
		m.data.RemoveRooms(ctx, toRemoveSlice)
	}
	if len(mapping) > 0 {
		log.Info().Int("rooms", len(mapping)).Msg("recreating room mappings...")
		if err := m.data.RecreateRoomMapping(ctx, mapping); err != nil {
			log.Error().Err(err).Msg("cannot recreate room mappings")
		}
	}

	// we put it here to ensure it will run only once in the full cycle
	m.removeOldOfflineServers(ctx)
}

// getServerContacts as per MSC1929
func (m *Crawler) getServerContacts(ctx context.Context, name string) model.MatrixServerContacts {
	var contacts model.MatrixServerContacts
	resp, err := msc1929.GetWithContext(ctx, name)
	if err != nil {
		return contacts
	}
	if resp.IsEmpty() {
		return contacts
	}

	if emails := resp.ModeratorEmails(); len(emails) > 0 {
		contacts.Emails = kit.Uniq(emails)
	} else if emails := resp.AdminEmails(); len(emails) > 0 {
		contacts.Emails = kit.Uniq(emails)
	} else {
		contacts.Emails = kit.Uniq(resp.AllEmails())
	}

	if mxids := resp.ModeratorMatrixIDs(); len(mxids) > 0 {
		contacts.MXIDs = kit.Uniq(mxids)
	} else if mxids := resp.AdminMatrixIDs(); len(mxids) > 0 {
		contacts.MXIDs = kit.Uniq(mxids)
	} else {
		contacts.MXIDs = kit.Uniq(resp.AllMatrixIDs())
	}
	contacts.URL = resp.SupportPage
	return contacts
}

// getPublicRooms reads public rooms of the given server from the matrix client-server api
// and sends them into channel
//
//nolint:gocognit // TODO: refactor
func (m *Crawler) getPublicRooms(ctx context.Context, name string) *kit.List[string, string] {
	var since string
	var added int
	limit := "10000"
	servers := kit.NewList[string, string]()
	log := apm.Log(ctx)

	for {
		start := time.Now()
		resp, err := m.fed.QueryPublicRooms(ctx, name, limit, since)
		if err != nil {
			log.Warn().Err(err).Str("server", name).Msg("cannot query public rooms")
			return servers
		}
		if len(resp.Chunk) == 0 {
			log.Info().Str("server", name).Msg("no public rooms available")
			return servers
		}

		added += len(resp.Chunk)
		for _, rdRoom := range resp.Chunk {
			room := rdRoom.Convert()
			if !m.v.IsRoomAllowed(name, room) {
				added--
				continue
			}

			if !room.Parse(m.detector, m.media, m.cfg.Get().Matrix.ServerName) {
				added--
				continue
			}

			qDir, err := m.fed.QueryDirectoryExternal(ctx, room.Alias)
			if err != nil {
				log.Warn().Err(err).Str("server", name).Str("room", room.ID).Str("alias", room.Alias).Msg("cannot query client directory")
			}
			if qDir != nil && len(qDir.Servers) > 0 {
				room.Server = strings.Join(kit.Uniq(append(room.Servers(), qDir.Servers...)), ",")
			}

			servers.AddSlice(room.Servers())

			m.data.AddRoomBatch(ctx, room)
			m.data.AddRoomMapping(ctx, room.ID, room.Alias) //nolint:errcheck // ignore error
		}
		log.
			Info().
			Str("server", name).
			Int("added", added).
			Int("of", resp.Total).
			Str("took", time.Since(start).String()).
			Msg("added rooms")

		if resp.NextBatch == "" {
			return servers
		}

		since = resp.NextBatch
	}
}
