package data

import (
	"context"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog"
	"go.etcd.io/bbolt"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

// AddServer info
func (d *Data) AddServer(ctx context.Context, server *model.MatrixServer) error {
	span := utils.StartSpan(ctx, "data.AddServer")
	defer span.Finish()
	log := zerolog.Ctx(ctx)

	return d.db.Batch(func(tx *bbolt.Tx) error {
		serverb, merr := json.Marshal(server)
		if merr != nil {
			log.Error().Err(merr).Str("server", server.Name).Msg("cannot marshal server")
			return merr
		}
		return tx.Bucket(serversInfoBucket).Put([]byte(server.Name), serverb)
	})
}

// BatchServers adds a batch of servers at once
func (d *Data) BatchServers(ctx context.Context, servers []string) error {
	span := utils.StartSpan(ctx, "data.BatchServers")
	defer span.Finish()
	log := zerolog.Ctx(ctx)

	return d.db.Batch(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(serversInfoBucket)
		for _, server := range servers {
			if v := bucket.Get([]byte(server)); v == nil {
				v, merr := json.Marshal(&model.MatrixServer{Name: server})
				if merr != nil {
					log.Error().Err(merr).Msg("cannot marshal server")
					continue
				}
				if err := bucket.Put([]byte(server), v); err != nil {
					log.Error().Err(err).Msg("cannot store server")
				}
			}
		}
		return nil
	})
}

// HasServer checks if server is already exists
func (d *Data) HasServer(ctx context.Context, name string) bool {
	span := utils.StartSpan(ctx, "data.HasServer")
	defer span.Finish()

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
	span := utils.StartSpan(ctx, "data.GetServerInfo")
	defer span.Finish()

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
	span := utils.StartSpan(ctx, "data.RemoveServer")
	defer span.Finish()

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
func (d *Data) RemoveServers(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}

	span := utils.StartSpan(ctx, "data.RemoveServers")
	defer span.Finish()

	d.db.Update(func(tx *bbolt.Tx) error { //nolint:errcheck // that's ok
		sbucket := tx.Bucket(serversBucket)
		sibucket := tx.Bucket(serversInfoBucket)
		for _, k := range keys {
			sbucket.Delete([]byte(k))  //nolint:errcheck // that's ok
			sibucket.Delete([]byte(k)) //nolint:errcheck // that's ok
		}
		return nil
	})
}

func (d *Data) markServerOffline(ctx context.Context, bucket *bbolt.Bucket, name string) {
	log := zerolog.Ctx(ctx)
	var server *model.MatrixServer
	key := []byte(name)
	if v := bucket.Get(key); v != nil {
		if merr := json.Unmarshal(v, &server); merr != nil {
			log.Error().Err(merr).Msg("cannot unmarshal server")
		}
	}
	if server == nil {
		server = &model.MatrixServer{Name: name}
	}

	server.Online = false
	server.Indexable = false

	datab, merr := json.Marshal(server)
	if merr != nil {
		log.Error().Err(merr).Msg("cannot marshal server")
		return
	}

	if err := bucket.Put(key, datab); err != nil {
		log.Error().Err(err).Msg("cannot store server")
	}
}

// MarkServersOffline from db
func (d *Data) MarkServersOffline(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}

	span := utils.StartSpan(ctx, "data.MarkServersOffline")
	defer span.Finish()

	d.db.Batch(func(tx *bbolt.Tx) error { //nolint:errcheck // that's ok
		bucket := tx.Bucket(serversInfoBucket)
		for _, k := range keys {
			d.markServerOffline(ctx, bucket, k)
		}
		return nil
	})
}

func (d *Data) FilterServers(ctx context.Context, filter func(server *model.MatrixServer) bool) map[string]*model.MatrixServer {
	span := utils.StartSpan(ctx, "data.FilterServers")
	defer span.Finish()
	log := zerolog.Ctx(ctx)

	servers := make(map[string]*model.MatrixServer)
	err := d.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(serversInfoBucket).ForEach(func(k, v []byte) error {
			if v == nil {
				return nil
			}
			var server *model.MatrixServer
			if err := json.Unmarshal(v, &server); err != nil {
				log.Error().Err(err).Str("server", string(k)).Msg("cannot unmarshal server")
			}
			if filter(server) {
				servers[string(k)] = server
			}

			return nil
		})
	})
	if err != nil {
		log.Error().Err(err).Msg("cannot filter servers")
	}

	return servers
}
