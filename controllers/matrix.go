package controllers

import (
	"bytes"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

type matrixService interface {
	GetWellKnown() []byte
	GetVersion() []byte
	GetKeyServer() []byte
	PublicRooms(*http.Request, *model.RoomDirectoryRequest) (int, []byte)
	QueryDirectory(req *http.Request, alias string) (int, []byte)
	QueryClientDirectory(alias string) (int, []byte)
}

func configureMatrixEndpoints(e *echo.Echo, matrixSvc matrixService) {
	e.GET("/.well-known/matrix/server", func(c echo.Context) error { return c.JSONBlob(http.StatusOK, matrixSvc.GetWellKnown()) })
	e.GET("/_matrix/federation/v1/version", func(c echo.Context) error { return c.JSONBlob(http.StatusOK, matrixSvc.GetVersion()) })
	e.GET("/_matrix/key/v2/server", func(c echo.Context) error { return c.JSONBlob(http.StatusOK, matrixSvc.GetKeyServer()) })

	e.GET("/_matrix/federation/v1/query/directory", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.QueryDirectory(c.Request(), c.QueryParam("room_alias")))
	})
	e.GET("/_matrix/client/r0/directory/room/:room_alias", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.QueryClientDirectory(c.Param("room_alias")))
	})
	e.GET("/_matrix/client/v3/directory/room/:room_alias", func(c echo.Context) error {
		return c.JSONBlob(matrixSvc.QueryClientDirectory(c.Param("room_alias")))
	})

	e.GET("/_matrix/federation/v1/publicRooms", matrixRoomDirectory(matrixSvc))
	e.POST("/_matrix/federation/v1/publicRooms", matrixRoomDirectory(matrixSvc))
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
