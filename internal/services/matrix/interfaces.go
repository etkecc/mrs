package matrix

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/etkecc/mrs/internal/model"
)

type configService interface {
	Get() *model.Config
}

type searchService interface {
	Search(ctx context.Context, req *http.Request, query, sortBy string, roomTypes []string, limit, offset int) ([]*model.Entry, int, error)
}

type mediaService interface {
	Get(ctx context.Context, serverName, mediaID string, params url.Values) (content io.Reader, contentType string)
	Add(ctx context.Context, serverName, mediaID string, params url.Values, content []byte)
}

type blocklistService interface {
	ByServer(server string) bool
}

type dataRepository interface {
	GetRoom(ctx context.Context, roomID string) (*model.MatrixRoom, error)
	GetRoomMapping(ctx context.Context, roomIDorAlias string) string
	IsBanned(ctx context.Context, roomID string) bool
}
