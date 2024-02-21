package services

import (
	"context"
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
	Allowed(ctx context.Context, serverName, endpoint string) bool
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
	EachRoom(context.Context, func(string, *model.MatrixRoom) bool)
	SetBiggestRooms(context.Context, []string) error
	GetBannedRooms(context.Context, ...string) ([]string, error)
	RemoveRooms(context.Context, []string)
	BanRoom(context.Context, string) error
	UnbanRoom(context.Context, string) error
	GetReportedRooms(context.Context, ...string) (map[string]string, error)
	ReportRoom(context.Context, string, string) error
	UnreportRoom(context.Context, string) error
	IsReported(context.Context, string) bool
}

type ValidatorService interface {
	Domain(server string) bool
	IsOnline(ctx context.Context, server string) (string, bool)
	IsIndexable(ctx context.Context, server string) bool
	IsRoomAllowed(ctx context.Context, server string, room *model.MatrixRoom) bool
}

type FederationService interface {
	QueryPublicRooms(ctx context.Context, serverName, limit, since string) (*model.RoomDirectoryResponse, error)
	QueryServerName(ctx context.Context, serverName string) (string, error)
	QueryVersion(ctx context.Context, serverName string) (string, string, error)
	QueryCSURL(ctx context.Context, serverName string) string
}

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

// DiscoverServers across federation and remove invalid ones
func (m *Crawler) DiscoverServers(ctx context.Context, workers int, overrideList ...*utils.List[string, string]) {
	log := zerolog.Ctx(ctx)
	if m.discovering {
		log.Info().Msg("servers discovery already in progress, ignoring request")
		return
	}
	span := utils.StartSpan(ctx, "crawler.DiscoverServers")
	defer span.Finish()

	m.discovering = true
	defer func() { m.discovering = false }()

	var servers *utils.List[string, string]
	if len(overrideList) > 0 {
		servers = overrideList[0]
	} else {
		servers = m.loadServers(span.Context())
	}

	offline := m.discoverServers(span.Context(), servers, workers)

	log.Info().Int("offline", offline.Len()).Msg("marking offline servers")
	m.data.MarkServersOffline(span.Context(), offline.Slice())
}

// AddServers by name in bulk, intended for HTTP API
func (m *Crawler) AddServers(ctx context.Context, names []string, workers int) {
	span := utils.StartSpan(ctx, "crawler.AddServers")
	servers := utils.NewListFromSlice(names)
	// exclude already known servers first
	for _, server := range servers.Slice() {
		if m.data.HasServer(span.Context(), server) {
			servers.Remove(server)
		}
	}

	m.discoverServers(span.Context(), servers, workers)
}

// AddServer by name, intended for HTTP API
// returns http status code to send to the reporter
func (m *Crawler) AddServer(ctx context.Context, name string) int {
	span := utils.StartSpan(ctx, "crawler.AddServer")
	if m.data.HasServer(span.Context(), name) {
		return http.StatusAlreadyReported
	}

	server := m.discoverServer(span.Context(), name)
	if server == nil {
		return http.StatusUnprocessableEntity
	}

	return http.StatusCreated
}

// ParseRooms across all discovered servers
func (m *Crawler) ParseRooms(ctx context.Context, workers int) {
	log := zerolog.Ctx(ctx)
	if m.parsing {
		log.Info().Msg("room parsing already in progress, ignoring request")
		return
	}

	span := utils.StartSpan(ctx, "crawler.ParseRooms")
	defer span.Finish()

	m.parsing = true
	defer func() { m.parsing = false }()

	servers := utils.NewList[string, string]()
	servers.AddSlice(m.IndexableServers(span.Context()))
	servers.RemoveSlice(m.block.Slice())
	slice := servers.Slice()
	total := len(slice)

	if total < workers {
		workers = total
	}
	wp := workpool.New(workers)
	discoveredServers := utils.NewList[string, string]()
	log.Info().Int("servers", total).Int("workers", workers).Msg("parsing rooms")
	for _, srvName := range slice {
		name := srvName
		wp.Do(func() error {
			serversFromRooms := m.getPublicRooms(span.Context(), name)
			discoveredServers.AddSlice(serversFromRooms.Slice())
			return nil
		})
	}

	go utils.PoolProgress(wp, func() {
		log.Info().Int("of", servers.Len()).Msg("parsing rooms in progress")
	})
	wp.Wait() //nolint:errcheck
	m.data.FlushRoomBatch(span.Context())
	discoveredServers.RemoveSlice(servers.Slice())
	log.
		Info().
		Int("of", servers.Len()).
		Int("discovered_servers", discoveredServers.Len()).
		Msg("parsing rooms has been finished")

	m.DiscoverServers(span.Context(), m.cfg.Get().Workers.Discovery, discoveredServers)

	m.afterRoomParsing(span.Context())
}

// EachRoom allows to work with each known room
func (m *Crawler) EachRoom(ctx context.Context, handler func(roomID string, data *model.MatrixRoom) bool) {
	log := zerolog.Ctx(ctx)
	if m.eachrooming {
		log.Info().Msg("iterating over each room is already in progress, ignoring request")
		return
	}
	m.eachrooming = true
	defer func() { m.eachrooming = false }()

	toRemove := []string{}
	m.data.EachRoom(ctx, func(id string, room *model.MatrixRoom) bool {
		if !m.v.IsRoomAllowed(ctx, room.Server, room) {
			toRemove = append(toRemove, id)
			return false
		}

		return handler(id, room)
	})
	m.data.RemoveRooms(ctx, toRemove)
}

// OnlineServers returns all known online servers
func (m *Crawler) OnlineServers(ctx context.Context) []string {
	return utils.MapKeys(m.data.FilterServers(ctx, func(server *model.MatrixServer) bool {
		return server.Online
	}))
}

// IndexableServers returns all known indexable servers
func (m *Crawler) IndexableServers(ctx context.Context) []string {
	return utils.MapKeys(m.data.FilterServers(ctx, func(server *model.MatrixServer) bool {
		return server.Online && server.Indexable
	}))
}

func (m *Crawler) loadServers(ctx context.Context) *utils.List[string, string] {
	span := utils.StartSpan(ctx, "crawler.loadServers")
	defer span.Finish()

	log := zerolog.Ctx(span.Context())
	log.Info().Msg("loading servers")
	servers := utils.NewList[string, string]()
	servers.AddSlice(m.cfg.Get().Servers)
	log.Info().Int("servers", servers.Len()).Msg("loaded servers from config")
	servers.AddSlice(utils.MapKeys(m.data.FilterServers(span.Context(), func(_ *model.MatrixServer) bool {
		return true
	})))
	log.Info().Int("servers", servers.Len()).Msg("loaded servers from config and db")

	return servers
}

// discoverServer parses server information
func (m *Crawler) discoverServer(ctx context.Context, name string) *model.MatrixServer {
	span := utils.StartSpan(ctx, "crawler.discoverServer")
	defer span.Finish()

	name, ok := m.v.IsOnline(span.Context(), name)
	if name == "" {
		return nil
	}

	server := &model.MatrixServer{
		Name:     name,
		URL:      m.fed.QueryCSURL(span.Context(), name),
		Contacts: m.getServerContacts(span.Context(), name),
		Online:   ok,
		OnlineAt: time.Now().UTC(),
	}

	if m.v.IsIndexable(span.Context(), name) {
		server.Indexable = true
	}

	if err := m.data.AddServer(span.Context(), server); err != nil {
		zerolog.Ctx(span.Context()).
			Error().Err(err).Msg("cannot store server")
	}
	return server
}

// discoverServers parses servers information and returns lists of OFFLINE servers
func (m *Crawler) discoverServers(ctx context.Context, servers *utils.List[string, string], workers int) (offline *utils.List[string, string]) {
	wp := workpool.New(workers)
	log := zerolog.Ctx(ctx)
	online := utils.NewList[string, string]()
	offline = utils.NewList[string, string]()
	indexable := utils.NewList[string, string]() // just for stats
	log.Info().Int("servers", servers.Len()).Int("workers", workers).Msg("validating servers")

	for _, server := range servers.Slice() {
		srvName := server
		wp.Do(func() error {
			server := m.discoverServer(ctx, srvName)
			if server == nil {
				return nil
			}
			if server.Online {
				online.Add(server.Name)
			} else {
				offline.Add(server.Name)
			}
			if server.Indexable {
				indexable.Add(server.Name)
			}
			return nil
		})
	}
	go utils.PoolProgress(wp, func() {
		log.Info().
			Int("online", online.Len()).
			Int("offline", offline.Len()).
			Int("indexable", indexable.Len()).
			Int("of", servers.Len()).
			Msg("servers discovery in progress")
	})
	wp.Wait() //nolint:errcheck

	log.Info().
		Int("online", online.Len()).
		Int("offline", offline.Len()).
		Int("indexable", indexable.Len()).
		Int("of", servers.Len()).
		Msg("servers discovery finished")
	return offline
}

func (m *Crawler) afterRoomParsing(ctx context.Context) {
	type roomCount struct {
		id      string
		members int
	}

	span := utils.StartSpan(ctx, "crawler.afterRoomParsing")
	defer span.Finish()

	log := zerolog.Ctx(span.Context())
	log.Info().Msg("after room parsing......")
	started := time.Now().UTC()
	counts := []roomCount{}
	toRemove := []string{}
	m.data.EachRoom(span.Context(), func(id string, data *model.MatrixRoom) bool {
		if started.Sub(data.ParsedAt) >= 24*7*time.Hour { // parsed more than a week ago
			toRemove = append(toRemove, id)
			return false
		}

		counts = append(counts, roomCount{data.ID, data.Members})
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
	if err := m.data.SetBiggestRooms(span.Context(), ids); err != nil {
		log.Error().Err(err).Msg("cannot set biggest rooms")
	}
	log.Info().Str("took", time.Since(started).String()).Msg("biggest rooms have been calculated and stored")

	if len(toRemove) > 0 {
		log.Info().Int("rooms", len(toRemove)).Msg("removing rooms last updated more than a week ago...")
		m.data.RemoveRooms(span.Context(), toRemove)
	}
}

// getServerContacts as per MSC1929
func (m *Crawler) getServerContacts(ctx context.Context, name string) model.MatrixServerContacts {
	span := utils.StartSpan(ctx, "crawler.getServerContacts")
	defer span.Finish()

	var contacts model.MatrixServerContacts
	resp, err := msc1929.Get(name)
	if err != nil {
		return contacts
	}
	if resp.IsEmpty() {
		return contacts
	}

	contacts.Emails = utils.Uniq(resp.Emails())
	contacts.MXIDs = utils.Uniq(resp.MatrixIDs())
	contacts.URL = resp.SupportPage
	return contacts
}

// getPublicRooms reads public rooms of the given server from the matrix client-server api
// and sends them into channel
func (m *Crawler) getPublicRooms(ctx context.Context, name string) *utils.List[string, string] {
	var since string
	var added int
	limit := "10000"
	servers := utils.NewList[string, string]()
	withFakeAliases := m.cfg.Get().Experiments.FakeAliases
	span := utils.StartSpan(ctx, "crawler.getPublicRooms")
	defer span.Finish()
	log := zerolog.Ctx(span.Context())

	for {
		start := time.Now()
		resp, err := m.fed.QueryPublicRooms(span.Context(), name, limit, since)
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
			if !m.v.IsRoomAllowed(span.Context(), name, room) {
				added--
				continue
			}

			room.Parse(m.detector, m.cfg.Get().Public.API)
			if withFakeAliases {
				room.ParseAlias(m.cfg.Get().Matrix.ServerName)
			}
			servers.AddSlice(room.Servers(m.cfg.Get().Matrix.ServerName))

			m.data.AddRoomBatch(span.Context(), room)
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
