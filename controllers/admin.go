package controllers

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"

	"gitlab.com/etke.cc/mrs/api/utils"
)

type dataService interface {
	AddServer(context.Context, string) int
	AddServers(context.Context, []string, int)
	DiscoverServers(context.Context, int)
	ParseRooms(context.Context, int)
	Ingest(context.Context)
	Full(context.Context, int, int)
}

type crawlerService interface {
	OnlineServers(context.Context) []string
}

func servers(crawler crawlerService) echo.HandlerFunc {
	return func(c echo.Context) error {
		servers := crawler.OnlineServers(c.Request().Context())
		serversb, err := yaml.Marshal(servers)
		if err != nil {
			return err
		}
		return c.Blob(http.StatusOK, "application/x-yaml", serversb)
	}
}

func status(stats statsService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, stats.Get())
	}
}

func discover(data dataService, cfg configService) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		ctx = context.WithoutCancel(ctx)
		ctx = utils.NewContext(ctx)
		go data.DiscoverServers(ctx, cfg.Get().Workers.Discovery)
		return c.NoContent(http.StatusCreated)
	}
}

func parse(data dataService, cfg configService) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		ctx = context.WithoutCancel(ctx)
		ctx = utils.NewContext(ctx)
		go data.ParseRooms(ctx, cfg.Get().Workers.Parsing)
		return c.NoContent(http.StatusCreated)
	}
}

func reindex(data dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		ctx = context.WithoutCancel(ctx)
		ctx = utils.NewContext(ctx)
		go data.Ingest(ctx)
		return c.NoContent(http.StatusCreated)
	}
}

func full(data dataService, cfg configService) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		ctx = context.WithoutCancel(ctx)
		ctx = utils.NewContext(ctx)
		go data.Full(ctx, cfg.Get().Workers.Discovery, cfg.Get().Workers.Parsing)
		return c.NoContent(http.StatusCreated)
	}
}
