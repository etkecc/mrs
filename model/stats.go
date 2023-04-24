package model

import "time"

// IndexStats structure
type IndexStats struct {
	Servers   IndexStatsServers `json:"servers"`
	Rooms     int               `json:"rooms"`
	Discovery IndexStatsTime    `json:"discovery"`
	Parsing   IndexStatsTime    `json:"parsing"`
	Indexing  IndexStatsTime    `json:"indexing"`
}

// IndexStatsServers structure
type IndexStatsServers struct {
	All    int `json:"all"`
	Online int `json:"online"`
}

// IndexStatsTime structure
type IndexStatsTime struct {
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}
