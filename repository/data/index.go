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
		Discovery: model.IndexStatsTime{},
		Parsing:   model.IndexStatsTime{},
		Indexing:  model.IndexStatsTime{},
	}
	d.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(indexBucket)

		serversBytes := bucket.Get([]byte("servers"))
		serversOnlineBytes := bucket.Get([]byte("servers_online"))
		roomsBytes := bucket.Get([]byte("rooms"))

		discoveryStartedAt := bucket.Get([]byte("discovery_started_at"))
		discoveryFinishedAt := bucket.Get([]byte("discovery_finished_at"))

		parsingStartedAt := bucket.Get([]byte("parsing_started_at"))
		parsingFinishedAt := bucket.Get([]byte("parsing_finished_at"))

		indexStartedAt := bucket.Get([]byte("indexing_started_at"))
		indexFinishedAt := bucket.Get([]byte("indexing_finished_at"))

		stats.Servers.All, _ = strconv.Atoi(string(serversBytes))
		stats.Servers.Online, _ = strconv.Atoi(string(serversOnlineBytes))
		stats.Rooms, _ = strconv.Atoi(string(roomsBytes))
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

// SetIndexServers sets count of discovered servers
func (d *Data) SetIndexServers(servers int) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(servers))
		return tx.Bucket(indexBucket).Put([]byte("servers"), value)
	})
}

// SetIndexOnlineServers sets count of discovered online servers
func (d *Data) SetIndexOnlineServers(servers int) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(servers))
		return tx.Bucket(indexBucket).Put([]byte("servers_online"), value)
	})
}

// SetIndexRooms sets count of indexed rooms
func (d *Data) SetIndexRooms(rooms int) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(rooms))
		return tx.Bucket(indexBucket).Put([]byte("rooms"), value)
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
