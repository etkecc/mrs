package data

import "go.etcd.io/bbolt"

var (
	serversBucket = []byte(`servers`)
	roomsBucket   = []byte(`rooms`)

	buckets = [][]byte{serversBucket, roomsBucket}
)

func initBuckets(db *bbolt.DB) error {
	return db.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists(bucket)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
