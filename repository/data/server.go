package data

import "go.etcd.io/bbolt"

// AddServer info
func (d *Data) AddServer(name, url string) error {
	return d.db.Batch(func(tx *bbolt.Tx) error {
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

// RemoveServer info
func (d *Data) RemoveServer(name string) error {
	return d.db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(serversBucket).Delete([]byte(name))
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
