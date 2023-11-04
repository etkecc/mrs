package services

import (
	"time"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

type dataCrawlerService interface {
	DiscoverServers(int) error
	AddServer(string) int
	AddServers([]string, int)
	AllServers() map[string]string
	ParseRooms(int)
	GetBiggestRooms() []*model.MatrixRoom
	EachRoom(func(string, *model.MatrixRoom))
}

type dataIndexService interface {
	EmptyIndex() error
	RoomsBatch(roomID string, data *model.Entry) error
	IndexBatch() error
}

type dataSearchService interface {
	SetEmptyQueryResults(rooms []*model.MatrixRoom)
}

type dataStatsService interface {
	Get() *model.IndexStats
	SetStartedAt(string, time.Time)
	SetFinishedAt(string, time.Time)
	Collect()
	CollectServers(bool)
}

type dataCacheService interface {
	Purge()
}

// DataFacade wraps all data-related services to provide reusable API across all components of the system
type DataFacade struct {
	crawler dataCrawlerService
	index   dataIndexService
	search  dataSearchService
	stats   dataStatsService
	cache   dataCacheService
}

// NewDataFacade creates new data facade service
func NewDataFacade(
	crawler dataCrawlerService,
	index dataIndexService,
	search dataSearchService,
	stats dataStatsService,
	cache dataCacheService,
) *DataFacade {
	return &DataFacade{crawler, index, search, stats, cache}
}

// AddServer by name, intended for HTTP API
// returns http status code to send to the reporter
func (df *DataFacade) AddServer(name string) int {
	defer df.stats.CollectServers(true)
	return df.crawler.AddServer(name)
}

// AddServers by name in bulk, intended for HTTP API
func (df *DataFacade) AddServers(names []string, workers int) {
	df.crawler.AddServers(names, workers)

	df.stats.CollectServers(true)
}

// DiscoverServers matrix servers
func (df *DataFacade) DiscoverServers(workers int) {
	utils.Logger.Info().Msg("discovering matrix servers...")
	start := time.Now().UTC()
	df.stats.SetStartedAt("discovery", start)
	err := df.crawler.DiscoverServers(workers)
	df.stats.SetFinishedAt("discovery", time.Now().UTC())
	utils.Logger.Info().Err(err).Str("took", time.Since(start).String()).Msg("servers discovery has been finished")
}

// ParseRooms from discovered servers
func (df *DataFacade) ParseRooms(workers int) {
	utils.Logger.Info().Msg("parsing matrix rooms...")
	start := time.Now().UTC()
	df.stats.SetStartedAt("parsing", start)
	df.crawler.ParseRooms(workers)
	df.stats.SetFinishedAt("parsing", time.Now().UTC())
	utils.Logger.Info().Str("took", time.Since(start).String()).Msg("matrix rooms have been parsed")

	df.search.SetEmptyQueryResults(df.crawler.GetBiggestRooms())
}

// Ingest data into search index
func (df *DataFacade) Ingest() {
	utils.Logger.Info().Msg("indexing matrix rooms...")
	if err := df.index.EmptyIndex(); err != nil {
		utils.Logger.Error().Err(err).Msg("cannot create empty index")
	}
	start := time.Now().UTC()
	df.stats.SetStartedAt("indexing", start)
	df.crawler.EachRoom(func(roomID string, room *model.MatrixRoom) {
		if err := df.index.RoomsBatch(roomID, room.Entry()); err != nil {
			utils.Logger.Warn().Err(err).Str("id", room.ID).Msg("cannot add room to batch")
		}
	})
	if err := df.index.IndexBatch(); err != nil {
		utils.Logger.Warn().Err(err).Msg("indexing of the last batch failed")
	}
	df.stats.SetFinishedAt("indexing", time.Now().UTC())
	utils.Logger.Info().Str("took", time.Since(start).String()).Msg("matrix rooms have been indexed")

	utils.Logger.Info().Msg("purging cache...")
	df.cache.Purge()
	utils.Logger.Info().Msg("cache has been purged")
}

// Full data pipeline (discovery, parsing, indexing)
func (df *DataFacade) Full(discoveryWorkers, parsingWorkers int) {
	df.DiscoverServers(discoveryWorkers)
	df.ParseRooms(parsingWorkers)
	df.Ingest()

	utils.Logger.Info().Msg("collecting stats...")
	df.stats.Collect()
	utils.Logger.Info().Msg("stats have been collected")
}
