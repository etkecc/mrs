package data

import (
	"crypto/sha1"
	"os"
	"path"
	"strconv"

	"gitlab.com/etke.cc/int/mrs/model"
)

type Data struct {
	servers    *Store[string, string]
	rooms      []*Store[string, model.Entry]
	roomShards int
}

func New(dir string, roomShards int) (*Data, error) {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, err
	}
	servers, err := NewStore[string, string](path.Join(dir, "servers.yml"), true)
	if err != nil {
		return nil, err
	}
	rooms := make([]*Store[string, model.Entry], roomShards)
	for i := 0; i < roomShards; i++ {
		store, err := NewStore[string, model.Entry](path.Join(dir, "rooms-"+strconv.Itoa(i)+".yml"), true)
		if err != nil {
			return nil, err
		}
		rooms[i] = store
	}

	return &Data{servers, rooms, roomShards}, nil
}

// getRoomsShard is a very simple sharding implementation
func (d *Data) getRoomsShard(roomID string) *Store[string, model.Entry] {
	hash := sha1.Sum([]byte(roomID))
	return d.rooms[int(hash[17])%d.roomShards]
}

// AddServer info
func (d *Data) AddServer(name, url string) {
	d.servers.Add(name, url)
}

// GetServer info
func (d *Data) GetServer(name string) string {
	return d.servers.Get(name)
}

// AddRoom info
func (d *Data) AddRoom(roomID string, data model.Entry) {
	d.getRoomsShard(roomID).Add(roomID, data)
}

// GetRoom info
func (d *Data) GetRoom(roomID string) model.Entry {
	return d.getRoomsShard(roomID).Get(roomID)
}

// Close data repository
//nolint:errcheck
func (d *Data) Close() {
	d.servers.Close()
	for _, room := range d.rooms {
		room.Close()
	}
}
