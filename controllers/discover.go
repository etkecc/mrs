package controllers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

func addServer(dataSvc dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := dataSvc.AddServer(c.Param("name"))
		return c.NoContent(code)
	}
}

func addServers(dataSvc dataService, workers int) echo.HandlerFunc {
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

		go dataSvc.AddServers(servers, workers)
		return c.NoContent(http.StatusAccepted)
	}
}
