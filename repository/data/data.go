package data

import (
	"encoding/json"

	"gitlab.com/etke.cc/int/mrs/model"
	"go.etcd.io/bbolt"
)

type Data struct {
	db *bbolt.DB
}

func New(path string) (*Data, error) {
	db, err := bbolt.Open(path, 0o600, nil)
	if err != nil {
		return nil, err
	}
	err = initBuckets(db)
	if err != nil {
		return nil, err
	}

	return &Data{db}, nil
}

// AddServer info
func (d *Data) AddServer(name, url string) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(serversBucket).Put([]byte(name), []byte(url))
	})
}

// GetServer info
func (d *Data) GetServer(name string) (string, error) {
	var url string
	err := d.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket(serversBucket).Get([]byte(name))
		if v != nil {
			url = string(v)
		}

		return nil
	})
	return url, err
}

// AddRoom info
func (d *Data) AddRoom(roomID string, data model.MatrixRoom) error {
	datab, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return d.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsBucket).Put([]byte(roomID), datab)
	})
}

// GetRoom info
func (d *Data) GetRoom(roomID string) (model.MatrixRoom, error) {
	var room model.MatrixRoom
	err := d.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket(roomsBucket).Get([]byte(roomID))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &room)
	})
	return room, err
}

// Close data repository
func (d *Data) Close() error {
	return d.db.Close()
}
