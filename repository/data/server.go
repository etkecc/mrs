package data

import (
	"encoding/json"
	"log"

	"go.etcd.io/bbolt"

	"gitlab.com/etke.cc/mrs/api/model"
)

// AddServer info
func (d *Data) AddServer(server *model.MatrixServer) error {
	return d.db.Batch(func(tx *bbolt.Tx) error {
		err := tx.Bucket(serversBucket).Put([]byte(server.Name), []byte(server.URL))
		if err != nil {
			log.Println(server.Name, "cannot add server", err)
			return err
		}

		serverb, merr := json.Marshal(server)
		if merr != nil {
			log.Println(server.Name, "cannot marshal server", merr)
			return merr
		}
		return tx.Bucket(serversInfoBucket).Put([]byte(server.Name), serverb)
	})
}

// HasServer checks if server is already exists
func (d *Data) HasServer(name string) bool {
	var has bool
	d.db.View(func(tx *bbolt.Tx) error { //nolint:errcheck // that's ok
		v := tx.Bucket(serversBucket).Get([]byte(name))
		has = v != nil
		return nil
	})
	return has
}

// GetServer URL
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

// EachServerInfo retruns each known matrix server
func (d *Data) EachServerInfo(handler func(name string, data *model.MatrixServer)) {
	var server *model.MatrixServer
	d.db.View(func(tx *bbolt.Tx) error { //nolint:errcheck // that's ok
		return tx.Bucket(serversInfoBucket).ForEach(func(k, v []byte) error {
			err := json.Unmarshal(v, &server)
			if err != nil {
				return err
			}
			handler(string(k), server)
			return nil
		})
	})
}

// GetServerInfo
func (d *Data) GetServerInfo(name string) (*model.MatrixServer, error) {
	var server *model.MatrixServer
	err := d.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket(serversInfoBucket).Get([]byte(name))
		if v == nil {
			return nil
		}
		err := json.Unmarshal(v, &server)
		if err != nil {
			return err
		}

		return nil
	})
	return server, err
}

// RemoveServer info
func (d *Data) RemoveServer(name string) error {
	nameb := []byte(name)
	return d.db.Batch(func(tx *bbolt.Tx) error {
		err := tx.Bucket(serversBucket).Delete(nameb)
		if err != nil {
			return err
		}
		return tx.Bucket(serversInfoBucket).Delete(nameb)
	})
}

// RemoveServers from db
func (d *Data) RemoveServers(keys []string) {
	if len(keys) == 0 {
		return
	}

	d.db.Update(func(tx *bbolt.Tx) error { //nolint:errcheck
		sbucket := tx.Bucket(serversBucket)
		sibucket := tx.Bucket(serversInfoBucket)
		for _, k := range keys {
			sbucket.Delete([]byte(k))  //nolint:errcheck
			sibucket.Delete([]byte(k)) //nolint:errcheck
		}
		return nil
	})
}

// AllServers returns all known servers
//
//nolint:errcheck
func (d *Data) AllServers() map[string]string {
	servers := make(map[string]string)
	d.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(serversBucket).ForEach(func(k, v []byte) error {
			servers[string(k)] = string(v)
			return nil
		})
	})

	return servers
}
