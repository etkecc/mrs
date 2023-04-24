package controllers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

func addServer(matrix matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := matrix.AddServer(c.Param("name"))
		return c.NoContent(code)
	}
}

func addServers(matrix matrixService, workers int) echo.HandlerFunc {
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

		go matrix.AddServers(servers, workers)
		return c.NoContent(http.StatusAccepted)
	}
}
