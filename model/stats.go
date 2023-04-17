package model

import "time"

// IndexStats structure
type IndexStats struct {
	Servers   int            `json:"servers"`
	Rooms     int            `json:"rooms"`
	Discovery IndexStatsTime `json:"discovery"`
	Parsing   IndexStatsTime `json:"parsing"`
	Index     IndexStatsTime `json:"index"`
}

// IndexStatsTime structure
type IndexStatsTime struct {
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}
