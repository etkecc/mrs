package controllers

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"

	"gitlab.com/etke.cc/mrs/api/model"
)

type matrixService interface {
	GetWellKnown() any
	GetVersion() any
	GetKeyServer() []byte
	PublicRooms(*model.RoomDirectoryRequest) *model.RoomDirectoryResponse
}

func configureMatrixEndpoints(e *echo.Echo, matrixSvc matrixService) {
	e.GET("/.well-known/matrix/server", func(c echo.Context) error { return c.JSON(http.StatusOK, matrixSvc.GetWellKnown()) })
	e.GET("/_matrix/federation/v1/version", func(c echo.Context) error { return c.JSON(http.StatusOK, matrixSvc.GetVersion()) })
	e.GET("/_matrix/key/v2/server", func(c echo.Context) error { return c.JSONBlob(http.StatusOK, matrixSvc.GetKeyServer()) })

	e.GET("/_matrix/federation/v1/publicRooms", matrixRoomDirectory(matrixSvc))
	e.POST("/_matrix/federation/v1/publicRooms", matrixRoomDirectory(matrixSvc))
}

// /_matrix/federation/v1/publicRooms
// TODO: authentication of the requester
// TODO: handle params
// TODO: document in swagger
func matrixRoomDirectory(matrixSvc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := &model.RoomDirectoryRequest{}
		if err := c.Bind(req); err != nil {
			log.Println("directory request binding failed:", err)
		}
		log.Printf("room directory:\nModel: %+v\nGET params: %+v\nHeaders: %+v", req, c.QueryParams(), c.Request().Header)

		return c.JSON(http.StatusOK, matrixSvc.PublicRooms(req))
	}
}
