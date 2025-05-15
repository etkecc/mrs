package data

import "go.etcd.io/bbolt"

var (
	// servers_info bucket
	// contains information about servers
	serversInfoBucket = []byte(`servers_info`)
	// rooms bucket
	// contains information about rooms
	roomsBucket = []byte(`rooms`)
	// rooms_mappings bucket
	// contains mappings room_id <-> room_alias
	roomsMappingsBucket = []byte(`rooms_mappings`)
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

	buckets = [][]byte{serversInfoBucket, roomsBucket, biggestRoomsBucket, roomsBanlistBucket, roomsReportsBucket, roomsMappingsBucket, indexBucket, indexTLBucket}
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
