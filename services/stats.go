package services

import (
	"log"

	"gitlab.com/etke.cc/int/mrs/model"
)

// Stats service
type Stats struct {
	data       DataRepository
	collecting bool
	servers    int
	rooms      int
}

// NewStats service
func NewStats(data DataRepository) *Stats {
	return &Stats{data: data}
}

// GetServers returns count of known servers
func (s *Stats) GetServers() int {
	return s.servers
}

// GetRooms returns count of known rooms
func (s *Stats) GetRooms() int {
	return s.rooms
}

// Collect stats from repository
func (s *Stats) Collect() {
	if s.collecting {
		log.Println("stats collection already in progress, ignoring request")
		return
	}
	s.collecting = true
	defer func() { s.collecting = false }()

	var servers int
	s.data.EachServer(func(_, _ string) {
		servers++
	})
	s.servers = servers

	var rooms int
	s.data.EachRoom(func(_ string, _ model.MatrixRoom) {
		rooms++
	})
	s.rooms = rooms
}
