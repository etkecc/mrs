package data

import (
	"bytes"
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog"
	"go.etcd.io/bbolt"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

// AddRoomBatch info
//
//nolint:errcheck
func (d *Data) AddRoomBatch(ctx context.Context, room *model.MatrixRoom) {
	d.rb.Add(ctx, room)
}

// FlushRoomBatch to ensure nothing is left
func (d *Data) FlushRoomBatch(ctx context.Context) {
	d.rb.Flush(ctx)
}

func (d *Data) SetBiggestRooms(ctx context.Context, ids []string) error {
	span := sentry.StartSpan(ctx, "db.query", sentry.WithDescription("data.SetBiggestRooms"))
	defer span.Finish()

	return d.db.Update(func(tx *bbolt.Tx) error {
		if err := tx.DeleteBucket(biggestRoomsBucket); err != nil {
			return err
		}
		brBucket, cerr := tx.CreateBucket(biggestRoomsBucket)
		if cerr != nil {
			return cerr
		}
		rBucket := tx.Bucket(roomsBucket)

		for i, id := range ids {
			roomID := []byte(id)
			v := rBucket.Get(roomID)
			if v == nil {
				continue
			}
			k := []byte(fmt.Sprintf("%06d", i+1))
			if err := brBucket.Put(k, v); err != nil {
				return err
			}
		}

		return nil
	})
}

func (d *Data) GetBiggestRooms(ctx context.Context, limit, offset int) []*model.MatrixRoom {
	span := sentry.StartSpan(ctx, "db.query", sentry.WithDescription("data.GetBiggestRooms"))
	defer span.Finish()
	log := zerolog.Ctx(ctx)

	min := []byte(fmt.Sprintf("%06d", offset))
	max := []byte(fmt.Sprintf("%06d", limit))
	rooms := []*model.MatrixRoom{}

	d.db.View(func(tx *bbolt.Tx) error { //nolint:errcheck
		c := tx.Bucket(biggestRoomsBucket).Cursor()
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			var room *model.MatrixRoom
			err := json.Unmarshal(v, &room)
			if err != nil {
				log.Error().Err(err).Msg("cannot unmarshal a biggest room")
				return err
			}
			rooms = append(rooms, room)
		}
		return nil
	})
	return rooms
}

// GetRoom info
func (d *Data) GetRoom(ctx context.Context, roomID string) (*model.MatrixRoom, error) {
	span := sentry.StartSpan(ctx, "db.query", sentry.WithDescription("data.GetRoom"))
	defer span.Finish()

	var room *model.MatrixRoom
	err := d.db.View(func(tx *bbolt.Tx) error {
		v := tx.Bucket(roomsBucket).Get([]byte(roomID))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &room)
	})
	return room, err
}

// RemoveRooms from db
func (d *Data) RemoveRooms(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}

	span := sentry.StartSpan(ctx, "db.query", sentry.WithDescription("data.RemoveRooms"))
	defer span.Finish()

	d.db.Update(func(tx *bbolt.Tx) error { //nolint:errcheck
		bucket := tx.Bucket(roomsBucket)
		for _, k := range keys {
			bucket.Delete([]byte(k)) //nolint:errcheck
		}
		return nil
	})
}

// EachRoom allows to work with each known room
//
//nolint:errcheck
func (d *Data) EachRoom(ctx context.Context, handler func(roomID string, data *model.MatrixRoom) bool) {
	span := sentry.StartSpan(ctx, "db.query", sentry.WithDescription("data.EachRoom"))
	defer span.Finish()

	var room *model.MatrixRoom
	d.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsBucket).ForEach(func(k, v []byte) error {
			err := json.Unmarshal(v, &room)
			if err != nil {
				return err
			}
			// ignore banned rooms
			if tx.Bucket(roomsBanlistBucket).Get(k) != nil {
				return nil
			}

			if handler(string(k), room) {
				return fmt.Errorf("stop")
			}

			return nil
		})
	})
}

// GetBannedRooms returns full list of the banned rooms
func (d *Data) GetBannedRooms(ctx context.Context, serverName ...string) ([]string, error) {
	span := sentry.StartSpan(ctx, "db.query", sentry.WithDescription("data.GetBannedRooms"))
	defer span.Finish()

	var server string
	if len(serverName) > 0 {
		server = serverName[0]
	}
	list := []string{}
	err := d.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsBanlistBucket).ForEach(func(k, v []byte) error {
			roomID := string(k)
			if server != "" && utils.ServerFrom(roomID) != server {
				return nil
			}

			list = append(list, string(k))
			return nil
		})
	})
	return list, err
}

// BanRoom
func (d *Data) BanRoom(ctx context.Context, roomID string) error {
	span := sentry.StartSpan(ctx, "db.query", sentry.WithDescription("data.BanRoom"))
	defer span.Finish()

	return d.db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsBanlistBucket).Put([]byte(roomID), []byte(`true`))
	})
}

// UnbanRoom
func (d *Data) UnbanRoom(ctx context.Context, roomID string) error {
	span := sentry.StartSpan(ctx, "db.query", sentry.WithDescription("data.UnbanRoom"))
	defer span.Finish()

	return d.db.Batch(func(tx *bbolt.Tx) error {
		if err := tx.Bucket(roomsBanlistBucket).Delete([]byte(roomID)); err != nil {
			return err
		}
		return tx.Bucket(roomsReportsBucket).Delete([]byte(roomID))
	})
}

// GetReportedRooms returns full list of the banned rooms with reasons
func (d *Data) GetReportedRooms(ctx context.Context, serverName ...string) (map[string]string, error) {
	span := sentry.StartSpan(ctx, "db.query", sentry.WithDescription("data.GetReportedRooms"))
	defer span.Finish()

	var server string
	if len(serverName) > 0 {
		server = serverName[0]
	}
	data := map[string]string{}
	err := d.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsReportsBucket).ForEach(func(k, v []byte) error {
			roomID := string(k)
			if server != "" && utils.ServerFrom(roomID) != server {
				return nil
			}

			data[string(k)] = string(v)
			return nil
		})
	})
	return data, err
}

// IsReported returns true if room was already reported
func (d *Data) IsReported(ctx context.Context, roomID string) bool {
	span := sentry.StartSpan(ctx, "db.query", sentry.WithDescription("data.IsReported"))
	defer span.Finish()

	var reported bool
	d.db.View(func(tx *bbolt.Tx) error { //nolint:errcheck
		v := tx.Bucket(roomsReportsBucket).Get([]byte(roomID))
		reported = v != nil
		return nil
	})

	return reported
}

// ReportRoom
func (d *Data) ReportRoom(ctx context.Context, roomID, reason string) error {
	span := sentry.StartSpan(ctx, "db.query", sentry.WithDescription("data.ReportRoom"))
	defer span.Finish()

	return d.db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsReportsBucket).Put([]byte(roomID), []byte(reason))
	})
}

// UnreportRoom
func (d *Data) UnreportRoom(ctx context.Context, roomID string) error {
	span := sentry.StartSpan(ctx, "db.query", sentry.WithDescription("data.UnreportRoom"))
	defer span.Finish()

	return d.db.Batch(func(tx *bbolt.Tx) error {
		return tx.Bucket(roomsReportsBucket).Delete([]byte(roomID))
	})
}
