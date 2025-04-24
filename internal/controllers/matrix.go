package controllers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/etkecc/go-apm"
	"github.com/labstack/echo/v4"

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
	GetClientRoomSummary(ctx context.Context, roomAliasOrID string) (int, []byte)
	GetClientMediaThumbnail(ctx context.Context, serverName, mediaID string, params url.Values) (io.Reader, string)
	GetMediaThumbnail(ctx context.Context, serverName, mediaID string, params url.Values) (io.Reader, string)
	PublicRooms(context.Context, *http.Request, *model.RoomDirectoryRequest) (int, []byte)
	QueryDirectory(ctx context.Context, req *http.Request, alias string) (int, []byte)
}

func configureMatrixS2SEndpoints(e *echo.Echo, matrixSvc matrixService, cacheSvc cacheService) {
	e.GET("/.well-known/matrix/server", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetServerWellKnown())
	}, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/federation/v1/version", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetServerVersion())
	}, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/key/v2/server", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetKeyServer(c.Request().Context()))
	})
	e.GET("/_matrix/federation/v1/query/directory", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.QueryDirectory(c.Request().Context(), c.Request(), c.QueryParam("room_alias")))
	})
	e.GET("/_matrix/federation/v1/publicRooms", matrixRoomDirectory(matrixSvc), cacheSvc.MiddlewareSearch())
	e.POST("/_matrix/federation/v1/publicRooms", matrixRoomDirectory(matrixSvc), cacheSvc.MiddlewareSearch())
}

func configureMatrixCSEndpoints(e *echo.Echo, matrixSvc matrixService, cacheSvc cacheService) {
	rl := getRL(30)
	e.GET("/.well-known/matrix/client", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetClientWellKnown())
	}, cacheSvc.MiddlewareImmutable())
	e.GET("/.well-known/matrix/support", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetSupportWellKnown())
	}, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/client/versions", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetClientVersion())
	}, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/media/r0/thumbnail/:name/:id", avatar(matrixSvc), rl, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/media/v3/thumbnail/:name/:id", avatar(matrixSvc), rl, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/client/r0/directory/room/:room_alias", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientDirectory(c.Request().Context(), c.Param("room_alias")))
	}, rl)
	e.GET("/_matrix/client/v3/directory/room/:room_alias", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientDirectory(c.Request().Context(), c.Param("room_alias")))
	}, rl)
	e.GET("/_matrix/client/r0/directory/list/room/:room_id", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientRoomVisibility(c.Request().Context(), c.Param("room_id")))
	}, rl, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/client/v3/directory/list/room/:room_id", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientRoomVisibility(c.Request().Context(), c.Param("room_id")))
	}, rl, cacheSvc.MiddlewareImmutable())

	// MSC3326 - correct and incorrect (but implemented by matrix.to) endpoints
	e.GET("/_matrix/client/unstable/im.nheko.summary/summary/:room_id_alias", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientRoomSummary(c.Request().Context(), c.Param("room_id_alias")))
	}, rl)
	e.GET("_matrix/client/unstable/im.nheko.summary/rooms/:room_id_alias/summary", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientRoomSummary(c.Request().Context(), c.Param("room_id_alias")))
	}, rl)
}

// /_matrix/federation/v1/publicRooms
func matrixRoomDirectory(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := apm.Log(c.Request().Context())
		r := c.Request()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		c.SetRequest(r)

		var req model.RoomDirectoryRequest
		if err := c.Bind(&req); err != nil {
			log.Error().Err(err).Msg("POST directory request binding failed")
		}
		req.IP = c.RealIP()
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		c.SetRequest(r)

		return c.JSONBlob(matrixSvc.PublicRooms(c.Request().Context(), c.Request(), &req))
	}
}
