package controllers

import (
	"bytes"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

type matrixService interface {
	GetServerWellKnown() []byte
	GetClientWellKnown() []byte
	GetServerVersion() []byte
	GetClientVersion() []byte
	GetKeyServer() []byte
	GetClientDirectory(alias string) (int, []byte)
	GetClientRoomVisibility(roomID string) (int, []byte)
	GetClientRoomSummary(roomAliasOrID string) (int, []byte)
	PublicRooms(*http.Request, *model.RoomDirectoryRequest) (int, []byte)
	QueryDirectory(req *http.Request, alias string) (int, []byte)
}

func configureMatrixS2SEndpoints(e *echo.Echo, matrixSvc matrixService, cacheSvc cacheService) {
	e.GET("/.well-known/matrix/server", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetServerWellKnown())
	}, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/federation/v1/version", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetServerVersion())
	}, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/key/v2/server", func(c echo.Context) error {
		return c.JSONBlob(http.StatusOK, matrixSvc.GetKeyServer())
	})
	e.GET("/_matrix/federation/v1/query/directory", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.QueryDirectory(c.Request(), c.QueryParam("room_alias")))
	})
	e.GET("/_matrix/federation/v1/publicRooms", matrixRoomDirectory(matrixSvc))
	e.POST("/_matrix/federation/v1/publicRooms", matrixRoomDirectory(matrixSvc))
}

func configureMatrixCSEndpoints(e *echo.Echo, matrixSvc matrixService, crawlerSvc crawlerService, cacheSvc cacheService) {
	rl := middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(1))
	e.GET("/.well-known/matrix/client", func(c echo.Context) error { return c.JSONBlob(http.StatusOK, matrixSvc.GetClientWellKnown()) })
	e.GET("/_matrix/client/versions", func(c echo.Context) error { return c.JSONBlob(http.StatusOK, matrixSvc.GetClientVersion()) })
	e.GET("/_matrix/media/r0/thumbnail/:name/:id", avatar(crawlerSvc), rl, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/media/v3/thumbnail/:name/:id", avatar(crawlerSvc), rl, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/client/r0/directory/room/:room_alias", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientDirectory(c.Param("room_alias")))
	}, rl)
	e.GET("/_matrix/client/v3/directory/room/:room_alias", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientDirectory(c.Param("room_alias")))
	}, rl)
	e.GET("/_matrix/client/r0/directory/list/room/:room_id", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientRoomVisibility(c.Param("room_id")))
	}, rl, cacheSvc.MiddlewareImmutable())
	e.GET("/_matrix/client/v3/directory/list/room/:room_id", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientRoomVisibility(c.Param("room_id")))
	}, rl, cacheSvc.MiddlewareImmutable())

	// MSC3326 - correct and incorrect (but implemented by matrix.to) endpoints
	e.GET("/_matrix/client/unstable/im.nheko.summary/summary/:room_id_alias", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientRoomSummary(c.Param("room_id_alias")))
	}, rl)
	e.GET("_matrix/client/unstable/im.nheko.summary/rooms/:room_id_alias/summary", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.GetClientRoomSummary(c.Param("room_id_alias")))
	}, rl)
}

// /_matrix/federation/v1/publicRooms
func matrixRoomDirectory(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := &model.RoomDirectoryRequest{}
		r := c.Request()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		c.SetRequest(r)

		if err := c.Bind(req); err != nil {
			utils.Logger.Error().Err(err).Msg("directory request binding failed")
		}
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		c.SetRequest(r)

		return c.JSONBlob(matrixSvc.PublicRooms(c.Request(), req))
	}
}
