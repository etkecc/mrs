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

// IndexStatsServers structure
type IndexStatsServers struct {
	Online    int `json:"online"`
	Indexable int `json:"indexable"`
	Blocked   int `json:"blocked"`
}

// IndexStatsRooms structure
type IndexStatsRooms struct {
	Indexed  int `json:"indexed"`
	Parsed   int `json:"parsed"`
	Banned   int `json:"banned"`
	Reported int `json:"reported"`
}

// IndexStatsTime structure
type IndexStatsTime struct {
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}
