package data

import "go.etcd.io/bbolt"

var (
	serversBucket = []byte(`servers`)
	roomsBucket   = []byte(`rooms`)
	indexBucket   = []byte(`index`)

	buckets = [][]byte{serversBucket, roomsBucket, indexBucket}
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
