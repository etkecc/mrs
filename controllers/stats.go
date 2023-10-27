package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func stats(stats statsService) echo.HandlerFunc {
	return func(c echo.Context) error {
		info := stats.Get()
		return c.JSON(http.StatusOK, map[string]int{
			"servers": info.Servers.Online,
			"rooms":   info.Rooms.Indexed,
		})
	}
}
