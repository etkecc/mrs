package controllers

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/model"
)

type dataService interface {
	AddServer(context.Context, string) int
	AddServers(context.Context, []string, int)
	DiscoverServers(context.Context, int)
	ParseRooms(context.Context, int)
	Ingest(context.Context)
	Full(context.Context, int, int)
	GetRoom(ctx context.Context, roomID string) (*model.MatrixRoom, error)
	EachRoom(context.Context, func(string, *model.MatrixRoom) bool)
}

// @Summary		Index status
// @Description	Full crawler and index statistics: server and room counts, plus the timing of the last discovery, parsing, and indexing passes. The admin-side twin of the public /stats, with more detail.
// @Tags			admin
// @Produce		json
// @Security		AdminAuth
// @Success		200	{object}	model.IndexStats
// @Router			/-/status [get]
func status(stats statsService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, stats.Get())
	}
}

// @Summary		Trigger discovery
// @Description	Kicks off a discovery pass and returns 201 immediately. Fire-and-forget: the crawl runs in the background, this does not wait for it.
// @Tags			admin
// @Produce		json
// @Security		AdminAuth
// @Success		201	"Discovery started in the background"
// @Router			/-/discover [post]
func discover(data dataService, cfg configService) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		ctx = context.WithoutCancel(ctx)
		go data.DiscoverServers(ctx, cfg.Get().Workers.Discovery)
		return c.NoContent(http.StatusCreated)
	}
}

// @Summary		Trigger parsing
// @Description	Kicks off a room-parsing pass and returns 201 immediately. Fire-and-forget, runs in the background.
// @Tags			admin
// @Produce		json
// @Security		AdminAuth
// @Success		201	"Parsing started in the background"
// @Router			/-/parse [post]
func parse(data dataService, cfg configService) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		ctx = context.WithoutCancel(ctx)
		go data.ParseRooms(ctx, cfg.Get().Workers.Parsing)
		return c.NoContent(http.StatusCreated)
	}
}

// @Summary		Trigger reindex
// @Description	Rebuilds the search index from what is already crawled and returns 201 immediately. Fire-and-forget, runs in the background.
// @Tags			admin
// @Produce		json
// @Security		AdminAuth
// @Success		201	"Reindex started in the background"
// @Router			/-/reindex [post]
func reindex(data dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		ctx = context.WithoutCancel(ctx)
		go data.Ingest(ctx)
		return c.NoContent(http.StatusCreated)
	}
}

// @Summary		Trigger a full cycle
// @Description	Runs a full cycle (discovery, then parsing) in one go and returns 201 immediately. Fire-and-forget, runs in the background. This is what a periodic cron/timer should hit to keep the index fresh.
// @Tags			admin
// @Produce		json
// @Security		AdminAuth
// @Success		201	"Full cycle started in the background"
// @Router			/-/full [post]
func full(data dataService, cfg configService) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		ctx = context.WithoutCancel(ctx)
		go data.Full(ctx, cfg.Get().Workers.Discovery, cfg.Get().Workers.Parsing)
		return c.NoContent(http.StatusCreated)
	}
}
