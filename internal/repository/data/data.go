package data

import (
	"context"

	"github.com/etkecc/go-apm"
	"github.com/goccy/go-json"
	"go.etcd.io/bbolt"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/repository/batch"
)

type Data struct {
	db *bbolt.DB
	rb *batch.Batch[*model.MatrixRoom]
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

	return &Data{
		db: db,
		rb: batch.New(10000, func(ctx context.Context, rooms []*model.MatrixRoom) {
			db.Update(func(tx *bbolt.Tx) error { //nolint:errcheck // checked inside
				log := apm.Log(ctx)
				for _, room := range rooms {
					roomb, err := json.Marshal(room)
					if err != nil {
						log.Error().Err(err).Str("id", room.ID).Str("server", room.Server).Msg("cannot marshal room")
						continue
					}

					err = tx.Bucket(roomsBucket).Put([]byte(room.ID), roomb)
					if err != nil {
						log.Error().Err(err).Str("id", room.ID).Str("server", room.Server).Msg("cannot add room")
						continue
					}
				}
				return nil
			})
		}),
	}, nil
}

// Close data repository
func (d *Data) Close() error {
	return d.db.Close()
}
