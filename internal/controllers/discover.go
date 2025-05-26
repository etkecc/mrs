package controllers

import (
	"context"
	"io"
	"net/http"

	"github.com/goccy/go-json"
	"github.com/labstack/echo/v4"
)

func addServer(dataSvc dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := dataSvc.AddServer(c.Request().Context(), c.Param("name"))
		return c.NoContent(code)
	}
}

func addServers(dataSvc dataService, cfg configService) echo.HandlerFunc {
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

		ctx := context.WithoutCancel(c.Request().Context())
		go dataSvc.AddServers(ctx, servers, cfg.Get().Workers.Discovery)
		return c.NoContent(http.StatusAccepted)
	}
}
