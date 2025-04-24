package data

import (
	"context"
	"strconv"

	"github.com/etkecc/go-apm"
	"go.etcd.io/bbolt"
)

// SetServersRoomsCount sets the count of rooms for each server
func (d *Data) SetServersRoomsCount(ctx context.Context, data map[string]int) error {
	apm.Log(ctx).Info().Int("count", len(data)).Msg("updating servers rooms count")
	return d.db.Batch(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(serversRoomsCountBucket)
		for server, count := range data {
			if count < 100 { // we don't want to expose servers with less than 100 rooms
				continue
			}
			err := bucket.Put([]byte(server), []byte(strconv.Itoa(count)))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetServersRoomsCount returns the count of rooms for each server
func (d *Data) GetServersRoomsCount(ctx context.Context) map[string]int {
	log := apm.Log(ctx)
	data := make(map[string]int)
	d.db.View(func(tx *bbolt.Tx) error { //nolint:errcheck // we don't care about errors here
		bucket := tx.Bucket(serversRoomsCountBucket)
		return bucket.ForEach(func(k, v []byte) error {
			count, err := strconv.Atoi(string(v))
			if err != nil {
				log.Error().Err(err).Str("server", string(k)).Msg("failed to convert value to int")
				return nil
			}
			data[string(k)] = count
			return nil
		})
	})
	return data
}

// SaveServersRooms saves the rooms for each server
//
//nolint:gocognit // TODO
func (d *Data) SaveServersRooms(ctx context.Context, data map[string][]string) error {
	log := apm.Log(ctx)
	return d.db.Update(func(tx *bbolt.Tx) error {
		if err := tx.DeleteBucket(serversRoomsBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucket(serversRoomsBucket); err != nil {
			return err
		}

		srBucket := tx.Bucket(serversRoomsBucket)
		rBucket := tx.Bucket(roomsBucket)
		for server, roomIDs := range data {
			if len(roomIDs) == 0 {
				continue
			}
			subBucket, err := srBucket.CreateBucket([]byte(server))
			if err != nil {
				log.Error().Err(err).Str("server", server).Msg("failed to create servers_rooms subbucket")
				return nil
			}
			for _, roomID := range roomIDs {
				if err := subBucket.Put([]byte(roomID), rBucket.Get([]byte(roomID))); err != nil {
					log.Error().Err(err).Str("server", server).Str("room_id", roomID).Msg("failed to put room into servers_rooms subbucket")
					return nil
				}
			}
		}
		return nil
	})
}
