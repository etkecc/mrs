package data

import (
	"bytes"
	"context"
	"fmt"

	"github.com/etkecc/go-apm"
	"github.com/goccy/go-json"
	"go.etcd.io/bbolt"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

// AddRoomBatch info
func (d *Data) AddRoomBatch(ctx context.Context, room *model.MatrixRoom) {
	d.rb.Add(ctx, room)
}

// FlushRoomBatch to ensure nothing is left
func (d *Data) FlushRoomBatch(ctx context.Context) {
	d.rb.Flush(ctx)
}

func (d *Data) SetBiggestRooms(ctx context.Context, ids []string) error {
	apm.Log(ctx).Info().Int("count", len(ids)).Msg("updating biggest rooms")

	return d.db.Update(func(tx *bbolt.Tx) error {
		if err := tx.DeleteBucket(biggestRoomsBucket); err != nil {
			return err
		}
		brBucket, cerr := tx.CreateBucket(biggestRoomsBucket)
		if cerr != nil {
			return cerr
		}
		rBucket := tx.Bucket(roomsBucket)

		for i, id := range ids {
			roomID := []byte(id)
			v := rBucket.Get(roomID)
			if v == nil {
				continue
			}
			k := []byte(fmt.Sprintf("%06d", i+1))
			if err := brBucket.Put(k, v); err != nil {
				return err
			}
		}

		return nil
	})
}

func (d *Data) GetBiggestRooms(ctx context.Context, limit, offset int) []*model.MatrixRoom {
	log := apm.Log(ctx)

	start := []byte(fmt.Sprintf("%06d", offset))
	end := []byte(fmt.Sprintf("%06d", limit))
	rooms := []*model.MatrixRoom{}

	d.db.View(func(tx *bbolt.Tx) error { //nolint:errcheck // that's ok
		c := tx.Bucket(biggestRoomsBucket).Cursor()
		for k, v := c.Seek(start); k != nil && bytes.Compare(k, end) <= 0; k, v = c.Next() {
			var room *model.MatrixRoom
			err := json.Unmarshal(v, &room)
			if err != nil {
				log.Error().Err(err).Msg("cannot unmarshal a biggest room")
				return err
			}
			rooms = append(rooms, room)
		}
		return nil
	})
	return rooms
}

// GetRoom info
func (d *Data) GetRoom(ctx context.Context, roomID string) (*model.MatrixRoom, error) {
	apm.Log(ctx).Debug().Str("room_id", roomID).Msg("getting a room")
	var room *model.MatrixRoom
	err := d.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket(roomsBucket).Get([]byte(roomID))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &room)
	})
	return room, err
}

// RemoveRooms from db
func (d *Data) RemoveRooms(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}

	apm.Log(ctx).Info().Int("count", len(keys)).Msg("removing rooms from db")
	d.db.Update(func(tx *bbolt.Tx) error { //nolint:errcheck // that's ok
		bucket := tx.Bucket(roomsBucket)
		mbucket := tx.Bucket(roomsMappingsBucket)
		for _, k := range keys {
			bucket.Delete([]byte(k))  //nolint:errcheck // that's ok
			mbucket.Delete([]byte(k)) //nolint:errcheck // that's ok
		}
		return nil
	})
}

// EachRoom allows to work with each known room
//
//nolint:errcheck // that's ok
func (d *Data) EachRoom(ctx context.Context, handler func(roomID string, data *model.MatrixRoom) bool) {
	apm.Log(ctx).Warn().Msg("iterating over all rooms")

	var room *model.MatrixRoom
	d.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsBucket).ForEach(func(k, v []byte) error {
			err := json.Unmarshal(v, &room)
			if err != nil {
				return err
			}
			// ignore banned rooms
			if tx.Bucket(roomsBanlistBucket).Get(k) != nil {
				return nil
			}

			if handler(string(k), room) {
				return fmt.Errorf("stop")
			}

			return nil
		})
	})
}

// GetBannedRooms returns full list of the banned rooms
func (d *Data) GetBannedRooms(ctx context.Context, serverName ...string) ([]string, error) {
	var server string
	if len(serverName) > 0 {
		server = serverName[0]
	}
	apm.Log(ctx).Info().Str("server", server).Msg("getting a list of banned rooms")
	list := []string{}
	err := d.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsBanlistBucket).ForEach(func(k, _ []byte) error {
			roomID := string(k)
			if server != "" && utils.ServerFrom(roomID) != server {
				return nil
			}

			list = append(list, string(k))
			return nil
		})
	})
	return list, err
}

// BanRoom
func (d *Data) BanRoom(ctx context.Context, roomID string) error {
	apm.Log(ctx).Info().Str("room_id", roomID).Msg("banning a room")
	return d.db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsBanlistBucket).Put([]byte(roomID), []byte(`true`))
	})
}

// UnbanRoom
func (d *Data) UnbanRoom(ctx context.Context, roomID string) error {
	apm.Log(ctx).Info().Str("room_id", roomID).Msg("unbanning a room")
	return d.db.Batch(func(tx *bbolt.Tx) error {
		if err := tx.Bucket(roomsBanlistBucket).Delete([]byte(roomID)); err != nil {
			return err
		}
		return tx.Bucket(roomsReportsBucket).Delete([]byte(roomID))
	})
}

// GetReportedRooms returns full list of the banned rooms with reasons
func (d *Data) GetReportedRooms(ctx context.Context, serverName ...string) (map[string]string, error) {
	apm.Log(ctx).Info().Msg("getting a list of reported rooms")

	var server string
	if len(serverName) > 0 {
		server = serverName[0]
	}
	data := map[string]string{}
	err := d.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsReportsBucket).ForEach(func(k, v []byte) error {
			roomID := string(k)
			if server != "" && utils.ServerFrom(roomID) != server {
				return nil
			}

			data[string(k)] = string(v)
			return nil
		})
	})
	return data, err
}

// IsReported returns true if room was already reported
func (d *Data) IsReported(ctx context.Context, roomID string) bool {
	apm.Log(ctx).Debug().Str("room_id", roomID).Msg("checking if a room is reported")
	var reported bool
	d.db.View(func(tx *bbolt.Tx) error { //nolint:errcheck // that's ok
		v := tx.Bucket(roomsReportsBucket).Get([]byte(roomID))
		reported = v != nil
		return nil
	})

	return reported
}

// ReportRoom
func (d *Data) ReportRoom(ctx context.Context, fromIP, roomID, reason string) error {
	apm.Log(ctx).Info().Str("room", roomID).Str("from", fromIP).Msg("reporting a room")
	return d.db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsReportsBucket).Put([]byte(roomID), []byte(reason))
	})
}

// UnreportRoom
func (d *Data) UnreportRoom(ctx context.Context, roomID string) error {
	apm.Log(ctx).Info().Str("room_id", roomID).Msg("unreporting a room")
	return d.db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsReportsBucket).Delete([]byte(roomID))
	})
}

func (d *Data) AddRoomMapping(ctx context.Context, roomID, alias string) error {
	if roomID == "" || alias == "" {
		return nil
	}

	apm.Log(ctx).Debug().Str("room_id", roomID).Str("alias", alias).Msg("adding a room mapping")
	return d.db.Batch(func(tx *bbolt.Tx) error {
		if err := tx.Bucket(roomsMappingsBucket).Put([]byte(roomID), []byte(alias)); err != nil {
			return err
		}
		return tx.Bucket(roomsMappingsBucket).Put([]byte(alias), []byte(roomID))
	})
}

// GetRoomMapping returns room ID or alias for a given room ID or alias
func (d *Data) GetRoomMapping(ctx context.Context, idOrAlias string) string {
	apm.Log(ctx).Debug().Str("id_or_alias", idOrAlias).Msg("getting a room mapping")
	var mapping string
	d.db.View(func(tx *bbolt.Tx) error { //nolint:errcheck // that's ok
		v := tx.Bucket(roomsMappingsBucket).Get([]byte(idOrAlias))
		if v != nil {
			mapping = string(v)
		}
		return nil
	})
	return mapping
}

// RemoveRoomMapping removes room ID and alias from the mapping
func (d *Data) RemoveRoomMapping(ctx context.Context, id, alias string) {
	apm.Log(ctx).Debug().Str("id", id).Str("alias", alias).Msg("removing a room mapping")
	d.db.Update(func(tx *bbolt.Tx) error { //nolint:errcheck // that's ok
		if id != "" {
			tx.Bucket(roomsMappingsBucket).Delete([]byte(id)) //nolint:errcheck // that's ok
		}
		if alias != "" {
			tx.Bucket(roomsMappingsBucket).Delete([]byte(alias)) //nolint:errcheck // that's ok
		}
		return nil
	})
}

// RecreateRoomMapping replaces the whole mapping with a new one
func (d *Data) RecreateRoomMapping(ctx context.Context, newMapping map[string]string) error {
	log := apm.Log(ctx)
	log.Info().Int("new_count", len(newMapping)).Msg("recreating room mappings")
	return d.db.Update(func(tx *bbolt.Tx) error {
		if err := tx.DeleteBucket(roomsMappingsBucket); err != nil {
			return err
		}
		mBucket, cerr := tx.CreateBucket(roomsMappingsBucket)
		if cerr != nil {
			return cerr
		}

		for k, v := range newMapping {
			if err := mBucket.Put([]byte(k), []byte(v)); err != nil {
				log.Error().Err(err).Str("key", k).Str("value", v).Msg("cannot put a room mapping")
			}
			if err := mBucket.Put([]byte(v), []byte(k)); err != nil {
				log.Error().Err(err).Str("key", v).Str("value", k).Msg("cannot put a room mapping")
			}
		}

		return nil
	})
}
