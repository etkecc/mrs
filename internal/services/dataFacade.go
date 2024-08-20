package services

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

type dataCrawlerService interface {
	DiscoverServers(context.Context, int, ...*utils.List[string, string])
	AddServer(context.Context, string) int
	AddServers(context.Context, []string, int)
	ParseRooms(context.Context, int)
	EachRoom(context.Context, func(string, *model.MatrixRoom) bool)
	GetServersRoomsCount(ctx context.Context) map[string]int
}

type dataIndexService interface {
	EmptyIndex(ctx context.Context) error
	RoomsBatch(ctx context.Context, roomID string, data *model.Entry) error
	IndexBatch(ctx context.Context) error
}

type dataStatsService interface {
	Get() *model.IndexStats
	SetStartedAt(context.Context, string, time.Time)
	SetFinishedAt(context.Context, string, time.Time)
	Collect(context.Context)
	CollectServers(context.Context, bool)
}

type dataCacheService interface {
	Purge(context.Context)
}

// DataFacade wraps all data-related services to provide reusable API across all components of the system
type DataFacade struct {
	crawler dataCrawlerService
	index   dataIndexService
	stats   dataStatsService
	cache   dataCacheService
}

// NewDataFacade creates new data facade service
func NewDataFacade(
	crawler dataCrawlerService,
	index dataIndexService,
	stats dataStatsService,
	cache dataCacheService,
) *DataFacade {
	return &DataFacade{crawler, index, stats, cache}
}

// AddServer by name, intended for HTTP API
// returns http status code to send to the reporter
func (df *DataFacade) AddServer(ctx context.Context, name string) int {
	defer df.stats.CollectServers(ctx, true)
	return df.crawler.AddServer(ctx, name)
}

// AddServers by name in bulk, intended for HTTP API
func (df *DataFacade) AddServers(ctx context.Context, names []string, workers int) {
	df.crawler.AddServers(ctx, names, workers)

	df.stats.CollectServers(ctx, true)
}

// DiscoverServers matrix servers
func (df *DataFacade) DiscoverServers(ctx context.Context, workers int) {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("discovering matrix servers...")

	start := time.Now().UTC()
	df.stats.SetStartedAt(ctx, "discovery", start)
	df.crawler.DiscoverServers(ctx, workers)
	df.stats.SetFinishedAt(ctx, "discovery", time.Now().UTC())
	log.Info().Str("took", time.Since(start).String()).Msg("servers discovery has been finished")
}

// ParseRooms from discovered servers
func (df *DataFacade) ParseRooms(ctx context.Context, workers int) {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("parsing matrix rooms...")
	start := time.Now().UTC()
	df.stats.SetStartedAt(ctx, "parsing", start)
	df.crawler.ParseRooms(ctx, workers)
	df.stats.SetFinishedAt(ctx, "parsing", time.Now().UTC())
	log.Info().Str("took", time.Since(start).String()).Msg("matrix rooms have been parsed")
}

// Ingest data into search index
func (df *DataFacade) Ingest(ctx context.Context) {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("creating fresh index...")
	if err := df.index.EmptyIndex(ctx); err != nil {
		log.Error().Err(err).Msg("cannot create empty index")
	}

	log.Info().Msg("indexing matrix rooms...")
	start := time.Now().UTC()
	df.stats.SetStartedAt(ctx, "indexing", start)
	df.crawler.EachRoom(ctx, func(roomID string, room *model.MatrixRoom) bool {
		if err := df.index.RoomsBatch(ctx, roomID, room.Entry()); err != nil {
			log.Warn().Err(err).Str("id", room.ID).Msg("cannot add room to batch")
		}
		return false
	})
	if err := df.index.IndexBatch(ctx); err != nil {
		log.Warn().Err(err).Msg("indexing of the last batch failed")
	}
	df.stats.SetFinishedAt(ctx, "indexing", time.Now().UTC())
	log.Info().Str("took", time.Since(start).String()).Msg("matrix rooms have been indexed")

	log.Info().Msg("purging cache...")
	df.cache.Purge(ctx)
	log.Info().Msg("cache has been purged")
}

// Full data pipeline (discovery, parsing, indexing)
func (df *DataFacade) Full(ctx context.Context, discoveryWorkers, parsingWorkers int) {
	span := utils.StartSpan(ctx, "dataFacade.Full")
	defer span.Finish()

	log := zerolog.Ctx(span.Context())
	df.DiscoverServers(span.Context(), discoveryWorkers)
	df.ParseRooms(span.Context(), parsingWorkers)
	df.Ingest(span.Context())

	log.Info().Msg("collecting stats...")
	df.stats.Collect(span.Context())
	log.Info().Msg("stats have been collected")
}

func (df *DataFacade) GetServersRoomsCount(ctx context.Context) map[string]int {
	return df.crawler.GetServersRoomsCount(ctx)
}
