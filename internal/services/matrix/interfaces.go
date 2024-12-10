package matrix

import (
	"context"

	"github.com/etkecc/mrs/internal/model"
)

type configService interface {
	Get() *model.Config
}

type searchService interface {
	Search(ctx context.Context, originServer, query, sortBy string, limit, offset int) ([]*model.Entry, int, error)
}

type dataRepository interface {
	EachRoom(ctx context.Context, handler func(roomID string, data *model.MatrixRoom) bool)
	GetRoom(ctx context.Context, roomID string) (*model.MatrixRoom, error)
}
