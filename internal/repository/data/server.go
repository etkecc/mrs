package data

import (
	"context"
	"time"

	"github.com/etkecc/go-apm/v2"
	"github.com/goccy/go-json"
	"go.etcd.io/bbolt"

	"github.com/etkecc/mrs/internal/model"
)

// AddServer info
func (d *Data) AddServer(ctx context.Context, server *model.MatrixServer) error {
	log := apm.Log(ctx)

	return d.db.Batch(func(tx *bbolt.Tx) error {
		serverb, merr := json.Marshal(server)
		if merr != nil {
			log.Error("cannot marshal server", "error", merr, "server", server.Name)
			return merr
		}
		return tx.Bucket(serversInfoBucket).Put([]byte(server.Name), serverb)
	})
}

// BatchServers adds a batch of servers at once
func (d *Data) BatchServers(ctx context.Context, servers []string) error {
	log := apm.Log(ctx)

	return d.db.Batch(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(serversInfoBucket)
		for _, server := range servers {
			if v := bucket.Get([]byte(server)); v == nil {
				// stamp OnlineAt, or the stub is year-1 and the same-run prune eats it on arrival.
				v, merr := json.Marshal(&model.MatrixServer{Name: server, OnlineAt: time.Now().UTC()})
				if merr != nil {
					log.Error("cannot marshal server", "error", merr)
					continue
				}
				if err := bucket.Put([]byte(server), v); err != nil {
					log.Error("cannot store server", "error", err)
				}
			}
		}
		return nil
	})
}

// HasServer checks if server is already exists
func (d *Data) HasServer(ctx context.Context, name string) bool {
	apm.Log(ctx).Debug("checking if server exists", "server", name)
	var has bool
	d.db.View(func(tx *bbolt.Tx) error { //nolint:errcheck // that's ok
		v := tx.Bucket(serversInfoBucket).Get([]byte(name))
		has = v != nil
		return nil
	})
	return has
}

// GetServerInfo
func (d *Data) GetServerInfo(ctx context.Context, name string) (*model.MatrixServer, error) {
	apm.Log(ctx).Debug("getting server info", "server", name)
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
func (d *Data) RemoveServer(ctx context.Context, name string) error {
	apm.Log(ctx).Debug("removing server info", "server", name)
	nameb := []byte(name)
	return d.db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(serversInfoBucket).Delete(nameb)
	})
}

// RemoveServers from db
func (d *Data) RemoveServers(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}

	apm.Log(ctx).Info("removing servers from db", "count", len(keys))
	d.db.Update(func(tx *bbolt.Tx) error { //nolint:errcheck // that's ok
		sibucket := tx.Bucket(serversInfoBucket)
		for _, k := range keys {
			sibucket.Delete([]byte(k)) //nolint:errcheck // that's ok
		}
		return nil
	})
}

func (d *Data) markServerOffline(ctx context.Context, bucket *bbolt.Bucket, name string) {
	log := apm.Log(ctx)
	var server *model.MatrixServer
	key := []byte(name)
	if v := bucket.Get(key); v != nil {
		if merr := json.Unmarshal(v, &server); merr != nil {
			log.Error("cannot unmarshal server", "error", merr)
		}
	}
	if server == nil {
		// dead on arrival: stamp OnlineAt now, or the stub lands in year-1 and this same run's prune buries it on sight.
		server = &model.MatrixServer{Name: name, OnlineAt: time.Now().UTC()}
	}

	server.Online = false
	server.Indexable = false
	server.CheckedAt = time.Now().UTC() // backoff clock; OnlineAt stays put or the 30d prune never fires and dead servers go immortal

	datab, merr := json.Marshal(server)
	if merr != nil {
		log.Error("cannot marshal server", "error", merr)
		return
	}

	if err := bucket.Put(key, datab); err != nil {
		log.Error("cannot store server", "error", err)
	}
}

// MarkServersOffline from db
func (d *Data) MarkServersOffline(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}

	apm.Log(ctx).Info("marking servers offline", "count", len(keys))
	d.db.Batch(func(tx *bbolt.Tx) error { //nolint:errcheck // that's ok
		bucket := tx.Bucket(serversInfoBucket)
		for _, k := range keys {
			if k == "" {
				continue
			}

			d.markServerOffline(ctx, bucket, k)
		}
		return nil
	})
}

func (d *Data) FilterServers(ctx context.Context, filter func(server *model.MatrixServer) bool) map[string]*model.MatrixServer {
	log := apm.Log(ctx)

	servers := make(map[string]*model.MatrixServer)
	err := d.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(serversInfoBucket).ForEach(func(k, v []byte) error {
			if v == nil {
				return nil
			}
			var server *model.MatrixServer
			if err := json.Unmarshal(v, &server); err != nil {
				log.Error("cannot unmarshal server", "error", err, "server", string(k))
				return nil
			}
			if filter(server) {
				servers[string(k)] = server
			}

			return nil
		})
	})
	if err != nil {
		log.Error("cannot filter servers", "error", err)
	}

	return servers
}
