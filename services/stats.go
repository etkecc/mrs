package services

import (
	"log"
	"time"

	"gitlab.com/etke.cc/mrs/api/model"
)

type StatsRepository interface {
	DataRepository
	GetIndexStats() *model.IndexStats
	SetIndexServers(servers int) error
	SetIndexRooms(rooms int) error
	SetStartedAt(process string, startedAt time.Time) error
	SetFinishedAt(process string, finishedAt time.Time) error
}

// Stats service
type Stats struct {
	data       StatsRepository
	stats      *model.IndexStats
	collecting bool
}

// NewStats service
func NewStats(data StatsRepository) *Stats {
	stats := &Stats{data: data}
	stats.Reload()

	return stats
}

// Reload saved stats. Useful when you need to get updated timestamps, but don't want to parse whole db
func (s *Stats) Reload() {
	s.stats = s.data.GetIndexStats()
}

// Get stats
func (s *Stats) Get() *model.IndexStats {
	return s.stats
}

// SetStartedAt of the process
func (s *Stats) SetStartedAt(process string, startedAt time.Time) {
	if err := s.data.SetStartedAt(process, startedAt); err != nil {
		log.Println("cannot set", process, "started_at", err)
	}
	s.stats = s.data.GetIndexStats()
}

// SetFinishedAt of the process
func (s *Stats) SetFinishedAt(process string, finishedAt time.Time) {
	if err := s.data.SetFinishedAt(process, finishedAt); err != nil {
		log.Println("cannot set", process, "finished_at", err)
	}
	s.stats = s.data.GetIndexStats()
}

// Collect stats from repository
func (s *Stats) Collect() {
	if s.collecting {
		log.Println("stats collection already in progress, ignoring request")
		return
	}
	s.collecting = true
	defer func() { s.collecting = false }()

	if err := s.data.SetIndexServers(len(s.data.AllServers())); err != nil {
		log.Println("cannot set indexed servers count", err)
	}

	var rooms int
	s.data.EachRoom(func(_ string, _ *model.MatrixRoom) {
		rooms++
	})
	if err := s.data.SetIndexRooms(rooms); err != nil {
		log.Println("cannot set indexed rooms count", err)
	}
	s.Reload()
}
