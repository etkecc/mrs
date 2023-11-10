package matrix

import "gitlab.com/etke.cc/mrs/api/model"

type configService interface {
	Get() *model.Config
}

type searchService interface {
	Search(query, sortBy string, limit, offset int) ([]*model.Entry, int, error)
}

type dataRepository interface {
	EachRoom(handler func(roomID string, data *model.MatrixRoom) bool)
	GetRoom(roomID string) (*model.MatrixRoom, error)
}
