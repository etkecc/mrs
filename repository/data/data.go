package data

import (
	"github.com/goccy/go-json"
	"go.etcd.io/bbolt"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/repository/batch"
	"gitlab.com/etke.cc/mrs/api/utils"
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
		rb: batch.New(10000, func(rooms []*model.MatrixRoom) {
			db.Update(func(tx *bbolt.Tx) error { //nolint:errcheck // checked inside
				for _, room := range rooms {
					roomb, err := json.Marshal(room)
					if err != nil {
						utils.Logger.Error().Err(err).Str("id", room.ID).Str("server", room.Server).Msg("cannot marshal room")
						continue
					}

					err = tx.Bucket(roomsBucket).Put([]byte(room.ID), roomb)
					if err != nil {
						utils.Logger.Error().Err(err).Str("id", room.ID).Str("server", room.Server).Msg("cannot add room")
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
