package services

import (
	"log"
	"time"

	"gitlab.com/etke.cc/mrs/api/model"
)

type dataMatrixService interface {
	DiscoverServers(int) error
	AddServer(string) int
	AllServers() map[string]string
	ParseRooms(int)
	EachRoom(func(string, *model.MatrixRoom))
}

type dataIndexService interface {
	RoomsBatch(roomID string, data *model.Entry) error
	IndexBatch() error
}

type dataStatsService interface {
	Get() *model.IndexStats
	SetStartedAt(string, time.Time)
	SetFinishedAt(string, time.Time)
	Collect()
}

type dataCacheService interface {
	Purge()
}

// DataFacade wraps all data-related services to provide reusable API across all components of the system
type DataFacade struct {
	matrix dataMatrixService
	index  dataIndexService
	stats  dataStatsService
	cache  dataCacheService
}

// NewDataFacade creates new data facade service
func NewDataFacade(
	matrix dataMatrixService,
	index dataIndexService,
	stats dataStatsService,
	cache dataCacheService,
) *DataFacade {
	return &DataFacade{matrix, index, stats, cache}
}

// DiscoverServers matrix servers
func (df *DataFacade) DiscoverServers(workers int) {
	log.Println("discovering matrix servers...")
	start := time.Now().UTC()
	df.stats.SetStartedAt("discovery", start)
	err := df.matrix.DiscoverServers(workers)
	df.stats.SetFinishedAt("discovery", time.Now().UTC())
	log.Println("servers discovery has been finished", err, "took", time.Since(start))
}

// ParseRooms from discovered servers
func (df *DataFacade) ParseRooms(workers int) {
	log.Println("parsing matrix rooms...")
	start := time.Now().UTC()
	df.stats.SetStartedAt("parsing", start)
	df.matrix.ParseRooms(workers)
	df.stats.SetFinishedAt("parsing", time.Now().UTC())
	log.Println("all available matrix rooms have been parsed; took", time.Since(start))
}

// Ingest data into search index
func (df *DataFacade) Ingest() {
	log.Println("ingesting matrix rooms...")
	start := time.Now().UTC()
	df.stats.SetStartedAt("indexing", start)
	df.matrix.EachRoom(func(roomID string, room *model.MatrixRoom) {
		if err := df.index.RoomsBatch(roomID, room.Entry()); err != nil {
			log.Println(room.Alias, "cannot add to batch", err)
		}
	})
	if err := df.index.IndexBatch(); err != nil {
		log.Println("indexing of the last batch failed", err)
	}
	df.stats.SetFinishedAt("indexing", time.Now().UTC())
	log.Println("all available matrix rooms have been ingested; took", time.Since(start))

	log.Println("purging cache...")
	df.cache.Purge()
	log.Println("cache has been purged")
}

// Full data pipeline (discovery, parsing, indexing)
func (df *DataFacade) Full(discoveryWorkers, parsingWorkers int) {
	df.DiscoverServers(discoveryWorkers)
	df.ParseRooms(parsingWorkers)
	df.Ingest()

	log.Println("collecting stats...")
	df.stats.Collect()
	log.Println("stats have been collected")
}
