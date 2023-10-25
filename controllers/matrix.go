package controllers

import (
	"bytes"
	"io"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"

	"gitlab.com/etke.cc/mrs/api/model"
)

type matrixService interface {
	GetWellKnown() []byte
	GetVersion() []byte
	GetKeyServer() []byte
	PublicRooms(*http.Request, *model.RoomDirectoryRequest) (int, []byte)
}

func configureMatrixEndpoints(e *echo.Echo, matrixSvc matrixService) {
	e.GET("/.well-known/matrix/server", func(c echo.Context) error { return c.JSONBlob(http.StatusOK, matrixSvc.GetWellKnown()) })
	e.GET("/_matrix/federation/v1/version", func(c echo.Context) error { return c.JSONBlob(http.StatusOK, matrixSvc.GetVersion()) })
	e.GET("/_matrix/key/v2/server", func(c echo.Context) error { return c.JSONBlob(http.StatusOK, matrixSvc.GetKeyServer()) })

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
			log.Println("directory request binding failed:", err)
		}
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		c.SetRequest(r)

		return c.JSONBlob(matrixSvc.PublicRooms(c.Request(), req))
	}
}
