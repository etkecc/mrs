package controllers

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/etkecc/mrs/internal/model"
)

type matrixService interface {
	GetServerWellKnown() []byte
	GetClientWellKnown() []byte
	GetSupportWellKnown() []byte
	GetServerVersion() []byte
	GetClientVersion() []byte
	GetKeyServer(context.Context) []byte
	GetClientDirectory(ctx context.Context, alias string) (int, []byte)
	GetClientRoomVisibility(ctx context.Context, roomID string) (int, []byte)
	GetClientRoomSummary(ctx context.Context, roomAliasOrID, via string, onlyMSC3266 bool) (int, *model.RoomDirectoryRoom)
	GetClientMediaThumbnail(ctx context.Context, serverName, mediaID string, params url.Values) (io.Reader, string)
	GetMediaThumbnail(ctx context.Context, serverName, mediaID string, params url.Values) (io.Reader, string)
	PublicRooms(context.Context, *http.Request, *model.RoomDirectoryRequest) (int, []byte)
	QueryDirectory(ctx context.Context, req *http.Request, alias string) (int, []byte)
	QueryServerKeys(ctx context.Context, serverName string, validUntilTS int64) []byte
	QueryServersKeys(ctx context.Context, req *model.QueryServerKeysRequest, validUntilTS int64) []byte
}
