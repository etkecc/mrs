package data

import (
	"encoding/json"
	"log"

	"go.etcd.io/bbolt"

	"gitlab.com/etke.cc/mrs/api/model"
)

// AddRoomBatch info
//
//nolint:errcheck
func (d *Data) AddRoomBatch(ch chan *model.MatrixRoom) {
	d.db.Update(func(tx *bbolt.Tx) error {
		for room := range ch {
			if room == nil {
				continue
			}

			roomb, err := json.Marshal(room)
			if err != nil {
				log.Println(room.Server, room.ID, "cannot marshal room", err)
				continue
			}

			err = tx.Bucket(roomsBucket).Put([]byte(room.ID), roomb)
			if err != nil {
				log.Println(room.Server, room.ID, "cannot add room", err)
				continue
			}
		}
		return nil
	})
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

// EachRoom allows to work with each known room
//
//nolint:errcheck
func (d *Data) EachRoom(handler func(roomID string, data *model.MatrixRoom)) {
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

			handler(string(k), room)
			return nil
		})
	})
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
		return tx.Bucket(roomsBanlistBucket).Delete([]byte(roomID))
	})
}
