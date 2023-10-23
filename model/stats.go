package model

import "time"

// IndexStats structure
type IndexStats struct {
	Servers   IndexStatsServers `json:"servers"`
	Rooms     IndexStatsRooms   `json:"rooms"`
	Discovery IndexStatsTime    `json:"discovery"`
	Parsing   IndexStatsTime    `json:"parsing"`
	Indexing  IndexStatsTime    `json:"indexing"`
}

// Clone stats
func (s *IndexStats) Clone() *IndexStats {
	if s == nil {
		return nil
	}
	return &IndexStats{
		Servers: IndexStatsServers{
			Online:  s.Servers.Online,
			Blocked: s.Servers.Blocked,
		},
		Rooms: IndexStatsRooms{
			All:      s.Rooms.All,
			Banned:   s.Rooms.Banned,
			Reported: s.Rooms.Reported,
		},
		Discovery: IndexStatsTime{
			StartedAt:  s.Discovery.StartedAt,
			FinishedAt: s.Discovery.FinishedAt,
		},
		Parsing: IndexStatsTime{
			StartedAt:  s.Parsing.StartedAt,
			FinishedAt: s.Parsing.FinishedAt,
		},
		Indexing: IndexStatsTime{
			StartedAt:  s.Indexing.StartedAt,
			FinishedAt: s.Indexing.FinishedAt,
		},
	}
}

// IndexStatsServers structure
type IndexStatsServers struct {
	Online  int `json:"online"`
	Blocked int `json:"blocked"`
}

// IndexStatsRooms structure
type IndexStatsRooms struct {
	All      int `json:"all"`
	Banned   int `json:"banned"`
	Reported int `json:"reported"`
}

// IndexStatsTime structure
type IndexStatsTime struct {
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}
