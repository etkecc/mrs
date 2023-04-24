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

// AllOnlineServers returns all servers known to be online
//
//nolint:errcheck
func (d *Data) AllOnlineServers() map[string]string {
	servers := make(map[string]string)
	d.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(serversInfoBucket).ForEach(func(k, v []byte) error {
			if v == nil {
				return nil
			}
			var server *model.MatrixServer
			err := json.Unmarshal(v, &server)
			if err != nil {
				log.Println("cannot unmarshal server:", err)
				return nil
			}

			if server.Online {
				servers[string(k)] = server.URL
			}
			return nil
		})
	})

	return servers
}
