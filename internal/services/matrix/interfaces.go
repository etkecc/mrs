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
	Search(ctx context.Context, originServer, query, sortBy string, limit, offset int) ([]*model.Entry, int, error)
}

type mediaService interface {
	Get(ctx context.Context, serverName, mediaID string, params url.Values) (content io.Reader, contentType string)
	Add(ctx context.Context, serverName, mediaID string, params url.Values, content []byte)
}

type dataRepository interface {
	EachRoom(ctx context.Context, handler func(roomID string, data *model.MatrixRoom) bool)
	GetRoom(ctx context.Context, roomID string) (*model.MatrixRoom, error)
}

type plausibleService interface {
	TrackSearch(ctx context.Context, incomingReq *http.Request, ip, query string)
}
