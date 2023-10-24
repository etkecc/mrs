package controllers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

func addServer(crawler crawlerService) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := crawler.AddServer(c.Param("name"))
		return c.NoContent(code)
	}
}

func addServers(crawler crawlerService, workers int) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer c.Request().Body.Close()
		jsonb, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		var servers []string
		err = json.Unmarshal(jsonb, &servers)
		if err != nil {
			return err
		}

		go crawler.AddServers(servers, workers)
		return c.NoContent(http.StatusAccepted)
	}
}
