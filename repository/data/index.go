package data

import (
	"strconv"
	"time"

	"go.etcd.io/bbolt"

	"gitlab.com/etke.cc/mrs/api/model"
)

// GetIndexStats returns index stats
//
//nolint:errcheck
func (d *Data) GetIndexStats() *model.IndexStats {
	stats := &model.IndexStats{
		Servers:   model.IndexStatsServers{},
		Rooms:     model.IndexStatsRooms{},
		Discovery: model.IndexStatsTime{},
		Parsing:   model.IndexStatsTime{},
		Indexing:  model.IndexStatsTime{},
	}
	d.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(indexBucket)

		serversOnlineBytes := bucket.Get([]byte("servers_online"))
		serversIndexableBytes := bucket.Get([]byte("servers_indexable"))
		serversBlockedBytes := bucket.Get([]byte("servers_blocked"))
		roomsIndexedBytes := bucket.Get([]byte("rooms"))
		roomsParsedBytes := bucket.Get([]byte("rooms_parsed"))
		roomsBannedBytes := bucket.Get([]byte("rooms_banned"))
		roomsReportedBytes := bucket.Get([]byte("rooms_reported"))

		discoveryStartedAt := bucket.Get([]byte("discovery_started_at"))
		discoveryFinishedAt := bucket.Get([]byte("discovery_finished_at"))

		parsingStartedAt := bucket.Get([]byte("parsing_started_at"))
		parsingFinishedAt := bucket.Get([]byte("parsing_finished_at"))

		indexStartedAt := bucket.Get([]byte("indexing_started_at"))
		indexFinishedAt := bucket.Get([]byte("indexing_finished_at"))

		stats.Servers.Online, _ = strconv.Atoi(string(serversOnlineBytes))
		stats.Servers.Indexable, _ = strconv.Atoi(string(serversIndexableBytes))
		stats.Servers.Blocked, _ = strconv.Atoi(string(serversBlockedBytes))
		stats.Rooms.Indexed, _ = strconv.Atoi(string(roomsIndexedBytes))
		stats.Rooms.Parsed, _ = strconv.Atoi(string(roomsParsedBytes))
		stats.Rooms.Banned, _ = strconv.Atoi(string(roomsBannedBytes))
		stats.Rooms.Reported, _ = strconv.Atoi(string(roomsReportedBytes))
		stats.Discovery.StartedAt, _ = time.Parse(time.RFC3339, string(discoveryStartedAt))
		stats.Discovery.FinishedAt, _ = time.Parse(time.RFC3339, string(discoveryFinishedAt))
		stats.Parsing.StartedAt, _ = time.Parse(time.RFC3339, string(parsingStartedAt))
		stats.Parsing.FinishedAt, _ = time.Parse(time.RFC3339, string(parsingFinishedAt))
		stats.Indexing.StartedAt, _ = time.Parse(time.RFC3339, string(indexStartedAt))
		stats.Indexing.FinishedAt, _ = time.Parse(time.RFC3339, string(indexFinishedAt))
		return nil
	})

	return stats
}

// SetIndexOnlineServers sets count of discovered online servers
func (d *Data) SetIndexOnlineServers(servers int) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(servers))
		return tx.Bucket(indexBucket).Put([]byte("servers_online"), value)
	})
}

// SetIndexIndexableServers sets count of discovered indexable servers
func (d *Data) SetIndexIndexableServers(servers int) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(servers))
		return tx.Bucket(indexBucket).Put([]byte("servers_indexable"), value)
	})
}

// SetIndexBlockedServers sets count of discovered online servers
func (d *Data) SetIndexBlockedServers(servers int) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(servers))
		return tx.Bucket(indexBucket).Put([]byte("servers_blocked"), value)
	})
}

// SetIndexIndexedRooms sets count of indexed rooms
func (d *Data) SetIndexIndexedRooms(rooms int) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(rooms))
		return tx.Bucket(indexBucket).Put([]byte("rooms"), value)
	})
}

// SetIndexParsedRooms sets count of parsed rooms
func (d *Data) SetIndexParsedRooms(rooms int) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(rooms))
		return tx.Bucket(indexBucket).Put([]byte("rooms_parsed"), value)
	})
}

// SetIndexBannedRooms sets count of banned rooms
func (d *Data) SetIndexBannedRooms(rooms int) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(rooms))
		return tx.Bucket(indexBucket).Put([]byte("rooms_banned"), value)
	})
}

// SetIndexReportedRooms sets count of banned rooms
func (d *Data) SetIndexReportedRooms(rooms int) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(rooms))
		return tx.Bucket(indexBucket).Put([]byte("rooms_reported"), value)
	})
}

// SetStartedAt sets start time of the new process
func (d *Data) SetStartedAt(process string, startedAt time.Time) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(startedAt.Format(time.RFC3339))
		return tx.Bucket(indexBucket).Put([]byte(process+"_started_at"), value)
	})
}

// SetFinishedAt sets end time of the finished process
func (d *Data) SetFinishedAt(process string, finishedAt time.Time) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(finishedAt.Format(time.RFC3339))
		return tx.Bucket(indexBucket).Put([]byte(process+"_finished_at"), value)
	})
}
