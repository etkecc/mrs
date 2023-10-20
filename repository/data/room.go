package data

import (
	"encoding/json"

	"go.etcd.io/bbolt"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

// AddRoomBatch info
//
//nolint:errcheck
func (d *Data) AddRoomBatch(room *model.MatrixRoom) {
	d.rb.Add(room)
}

// FlushRoomBatch to ensure nothing is left
func (d *Data) FlushRoomBatch() {
	d.rb.Flush()
}

// GetRoom info
func (d *Data) GetRoom(roomID string) (*model.MatrixRoom, error) {
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

func (d *Data) removeRooms(keys [][]byte) {
	if len(keys) == 0 {
		return
	}

	d.db.Update(func(tx *bbolt.Tx) error { //nolint:errcheck
		bucket := tx.Bucket(roomsBucket)
		for _, k := range keys {
			bucket.Delete(k) //nolint:errcheck
		}
		return nil
	})
}

// EachRoom allows to work with each known room
//
//nolint:errcheck
func (d *Data) EachRoom(blocklist []string, handler func(roomID string, data *model.MatrixRoom)) {
	var room *model.MatrixRoom
	toRemove := [][]byte{}
	d.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsBucket).ForEach(func(k, v []byte) error {
			if utils.ServerIn(blocklist, string(k)) {
				toRemove = append(toRemove, k)
				return nil
			}

			err := json.Unmarshal(v, &room)
			if err != nil {
				return err
			}
			// ignore banned rooms
			if tx.Bucket(roomsBanlistBucket).Get(k) != nil {
				toRemove = append(toRemove, k)
				return nil
			}

			if utils.ServerIn(blocklist, room.ID) || utils.ServerIn(blocklist, room.Alias) {
				toRemove = append(toRemove, k)
				return nil
			}

			handler(string(k), room)
			return nil
		})
	})
	d.removeRooms(toRemove)
}

// GetBannedRooms returns full list of the banned rooms
func (d *Data) GetBannedRooms(serverName ...string) ([]string, error) {
	var server string
	if len(serverName) > 0 {
		server = serverName[0]
	}
	list := []string{}
	err := d.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsBanlistBucket).ForEach(func(k, v []byte) error {
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
func (d *Data) BanRoom(roomID string) error {
	return d.db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsBanlistBucket).Put([]byte(roomID), []byte(`true`))
	})
}

// UnbanRoom
func (d *Data) UnbanRoom(roomID string) error {
	return d.db.Batch(func(tx *bbolt.Tx) error {
		if err := tx.Bucket(roomsBanlistBucket).Delete([]byte(roomID)); err != nil {
			return err
		}
		return tx.Bucket(roomsReportsBucket).Delete([]byte(roomID))
	})
}

// GetReportedRooms returns full list of the banned rooms with reasons
func (d *Data) GetReportedRooms(serverName ...string) (map[string]string, error) {
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
func (d *Data) IsReported(roomID string) bool {
	var reported bool
	d.db.View(func(tx *bbolt.Tx) error { //nolint:errcheck
		v := tx.Bucket(roomsReportsBucket).Get([]byte(roomID))
		reported = v != nil
		return nil
	})

	return reported
}

// ReportRoom
func (d *Data) ReportRoom(roomID, reason string) error {
	return d.db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsReportsBucket).Put([]byte(roomID), []byte(reason))
	})
}

// UnreportRoom
func (d *Data) UnreportRoom(roomID string) error {
	return d.db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsReportsBucket).Delete([]byte(roomID))
	})
}
