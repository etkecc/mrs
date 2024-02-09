package data

import "go.etcd.io/bbolt"

var (
	// servers bucket
	// Deprecated: use servers_info instead
	serversBucket = []byte(`servers`)
	// servers_info bucket
	// contains information about servers
	serversInfoBucket = []byte(`servers_info`)
	// rooms bucket
	// contains information about rooms
	roomsBucket = []byte(`rooms`)
	// biggest rooms bucket
	// contains the same content as rooms bucket, but sorted by the number of users
	biggestRoomsBucket = []byte(`rooms_biggest`)
	// rooms banlist bucket
	// contains information about banned rooms
	roomsBanlistBucket = []byte(`rooms_banlist`)
	// rooms reports bucket
	// contains information about reported rooms
	roomsReportsBucket = []byte(`rooms_reports`)
	// index bucket
	// contains latest index stats
	indexBucket = []byte(`index`)
	// index_timeline bucket
	// contains index stats by date
	indexTLBucket = []byte(`index_timeline`)

	buckets = [][]byte{serversBucket, serversInfoBucket, roomsBucket, biggestRoomsBucket, roomsBanlistBucket, roomsReportsBucket, indexBucket, indexTLBucket}
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
