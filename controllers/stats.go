package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func stats(stats statsService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]int{
			"servers": stats.GetServers(),
			"rooms":   stats.GetRooms(),
		})
	}
}
