package data

import (
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

// Close data repository
func (d *Data) Close() error {
	return d.db.Close()
}
