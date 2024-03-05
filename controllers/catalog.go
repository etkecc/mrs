package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func catalogServers(dataSvc dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, dataSvc.GetServersRoomsCount(c.Request().Context()))
	}
}
