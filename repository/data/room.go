package data

import (
	"encoding/json"

	"go.etcd.io/bbolt"

	"gitlab.com/etke.cc/mrs/api/model"
)

// AddRoom info
func (d *Data) AddRoom(roomID string, data *model.MatrixRoom) error {
	if data == nil {
		return nil
	}

	datab, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return d.db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsBucket).Put([]byte(roomID), datab)
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

			handler(string(k), room)
			return nil
		})
	})
}
