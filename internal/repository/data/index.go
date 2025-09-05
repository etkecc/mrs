package data

import (
	"bytes"
	"context"
	"strconv"
	"time"

	"github.com/etkecc/go-apm"
	"github.com/goccy/go-json"
	"go.etcd.io/bbolt"

	"github.com/etkecc/mrs/internal/model"
)

// SetIndexStatsTL sets index stats for the given time
func (d *Data) SetIndexStatsTL(ctx context.Context, calculatedAt time.Time, stats *model.IndexStats) error {
	apm.Log(ctx).Info().Msg("updating index stats")
	id := []byte(calculatedAt.UTC().Format(time.RFC3339))
	statsb, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	return d.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(indexTLBucket).Put(id, statsb)
	})
}

//nolint:gocognit // TODO: optimize the complexity
func (d *Data) getIndexStatsFullTL(ctx context.Context) (map[time.Time]*model.IndexStats, error) {
	apm.Log(ctx).Debug().Msg("getting index stats")
	months := map[string]struct{}{}
	weeks := map[int]struct{}{}
	currentYear := []byte(time.Now().UTC().Format("2006"))
	currentMonth := []byte(time.Now().UTC().Format("2006-01"))
	statsTL := make(map[time.Time]*model.IndexStats)
	err := d.db.View(func(tx *bbolt.Tx) error {
		var prev *model.IndexStats
		return tx.Bucket(indexTLBucket).ForEach(func(k, v []byte) error {
			if v == nil {
				return nil
			}
			t, err := time.Parse(time.RFC3339, string(k))
			if err != nil {
				return err
			}
			// if the result is for previous years,
			// keep only one result per month
			if !bytes.HasPrefix(k, currentYear) {
				month := t.Format("2006-01")
				if _, ok := months[month]; ok {
					return nil
				}
				months[month] = struct{}{}
			}

			// if the result is for the current year, but not for the current month,
			// keep only one result per week
			if bytes.HasPrefix(k, currentYear) && !bytes.HasPrefix(k, currentMonth) {
				_, week := t.ISOWeek()
				if _, ok := weeks[week]; ok {
					return nil
				}
				weeks[week] = struct{}{}
			}

			var stats *model.IndexStats
			if err := json.Unmarshal(v, &stats); err != nil {
				return err
			}
			// Ensure that there is no big difference between the current and previous stats,
			// because such difference may indicate a bug in the stats collection or a data corruption.
			if prev != nil {
				if stats.Rooms.Parsed > 5*prev.Rooms.Parsed || stats.Rooms.Parsed < prev.Rooms.Parsed/5 {
					return nil
				}
			}
			statsTL[t] = stats
			prev = stats
			return nil
		})
	})
	return statsTL, err
}

// GetIndexStatsTL returns index stats for the given time prefix in RFC3339 format
func (d *Data) GetIndexStatsTL(ctx context.Context, prefix string) (map[time.Time]*model.IndexStats, error) {
	apm.Log(ctx).Debug().Str("prefix", prefix).Msg("getting index stats timeline")
	if prefix == "" {
		return d.getIndexStatsFullTL(ctx)
	}
	statsTL := make(map[time.Time]*model.IndexStats)
	seek := []byte(prefix)
	err := d.db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket(indexTLBucket).Cursor()
		for k, v := c.Seek(seek); k != nil && bytes.HasPrefix(k, seek); k, v = c.Next() {
			if v == nil {
				continue
			}
			var stats *model.IndexStats
			if err := json.Unmarshal(v, &stats); err != nil {
				return err
			}
			t, err := time.Parse(time.RFC3339, string(k))
			if err != nil {
				return err
			}
			statsTL[t] = stats
		}
		return nil
	})

	return statsTL, err
}

// GetIndexStats returns index stats
//
//nolint:errcheck // that's ok
func (d *Data) GetIndexStats(ctx context.Context) *model.IndexStats {
	apm.Log(ctx).Debug().Msg("getting index stats")
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
		serversSoftwareBytes := bucket.Get([]byte("servers_software"))
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

		if err := json.Unmarshal(serversSoftwareBytes, &stats.Servers.Software); err != nil {
			stats.Servers.Software = map[string]int{}
		}

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
func (d *Data) SetIndexOnlineServers(ctx context.Context, servers int) error {
	apm.Log(ctx).Info().Int("count", servers).Msg("updating online servers count")
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(servers))
		return tx.Bucket(indexBucket).Put([]byte("servers_online"), value)
	})
}

// SetIndexIndexableServers sets count of discovered indexable servers
func (d *Data) SetIndexIndexableServers(ctx context.Context, servers int) error {
	apm.Log(ctx).Info().Int("count", servers).Msg("updating indexable servers count")
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(servers))
		return tx.Bucket(indexBucket).Put([]byte("servers_indexable"), value)
	})
}

// SetIndexBlockedServers sets count of discovered online servers
func (d *Data) SetIndexBlockedServers(ctx context.Context, servers int) error {
	apm.Log(ctx).Info().Int("count", servers).Msg("updating blocked servers count")
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(servers))
		return tx.Bucket(indexBucket).Put([]byte("servers_blocked"), value)
	})
}

// SetIndexServersSoftware sets map of discovered servers software
func (d *Data) SetIndexServersSoftware(ctx context.Context, software map[string]int) error {
	apm.Log(ctx).Info().Int("count", len(software)).Msg("updating servers software stats")
	return d.db.Update(func(tx *bbolt.Tx) error {
		value, err := json.Marshal(software)
		if err != nil {
			return err
		}
		return tx.Bucket(indexBucket).Put([]byte("servers_software"), value)
	})
}

// SetIndexIndexedRooms sets count of indexed rooms
func (d *Data) SetIndexIndexedRooms(ctx context.Context, rooms int) error {
	apm.Log(ctx).Info().Int("count", rooms).Msg("updating indexed rooms count")
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(rooms))
		return tx.Bucket(indexBucket).Put([]byte("rooms"), value)
	})
}

// SetIndexParsedRooms sets count of parsed rooms
func (d *Data) SetIndexParsedRooms(ctx context.Context, rooms int) error {
	apm.Log(ctx).Info().Int("count", rooms).Msg("updating parsed rooms count")
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(rooms))
		return tx.Bucket(indexBucket).Put([]byte("rooms_parsed"), value)
	})
}

// SetIndexBannedRooms sets count of banned rooms
func (d *Data) SetIndexBannedRooms(ctx context.Context, rooms int) error {
	apm.Log(ctx).Info().Int("count", rooms).Msg("updating banned rooms count")
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(rooms))
		return tx.Bucket(indexBucket).Put([]byte("rooms_banned"), value)
	})
}

// SetIndexReportedRooms sets count of banned rooms
func (d *Data) SetIndexReportedRooms(ctx context.Context, rooms int) error {
	apm.Log(ctx).Info().Int("count", rooms).Msg("updating reported rooms count")
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(strconv.Itoa(rooms))
		return tx.Bucket(indexBucket).Put([]byte("rooms_reported"), value)
	})
}

// SetStartedAt sets start time of the new process
func (d *Data) SetStartedAt(ctx context.Context, process string, startedAt time.Time) error {
	apm.Log(ctx).Info().Str("process", process).Msg("updating started at time")
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(startedAt.Format(time.RFC3339))
		return tx.Bucket(indexBucket).Put([]byte(process+"_started_at"), value)
	})
}

// SetFinishedAt sets end time of the finished process
func (d *Data) SetFinishedAt(ctx context.Context, process string, finishedAt time.Time) error {
	apm.Log(ctx).Info().Str("process", process).Msg("updating finished at time")
	return d.db.Update(func(tx *bbolt.Tx) error {
		value := []byte(finishedAt.Format(time.RFC3339))
		return tx.Bucket(indexBucket).Put([]byte(process+"_finished_at"), value)
	})
}
