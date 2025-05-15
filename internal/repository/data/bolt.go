package data

import (
	"go.etcd.io/bbolt"
)

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
	bucketsMap := map[string]bool{}
	for _, bucket := range buckets {
		bucketsMap[string(bucket)] = true
	}

	return db.Update(func(tx *bbolt.Tx) error {
		// Remove unused buckets
		if err := cleanupBuckets(bucketsMap, tx); err != nil {
			return err
		}

		// Create buckets if they don't exist
		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists(bucket)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func cleanupBuckets(bucketsMap map[string]bool, tx *bbolt.Tx) error {
	cursor := tx.Cursor()
	toRemove := [][]byte{}
	for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
		if !bucketsMap[string(k)] {
			toRemove = append(toRemove, k)
		}
	}
	for _, bucket := range toRemove {
		err := tx.DeleteBucket(bucket)
		if err != nil {
			return err
		}
	}
	return nil
}
