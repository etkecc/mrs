package controllers

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

type crawlerService interface {
	OnlineServers(context.Context) []string
	OnlineServersObjects(context.Context) map[string]*model.MatrixServer
}

// @Summary		Room preview
// @Description	Room preview by ID or alias: like the Matrix client-server room preview, but enriched with everything MRS knows about a room (language and so on). We try our own index first, then fall back to a live MSC3266 summary, and when that fallback fires we set the `X-MRS-MSC3266: true` response header so you can tell. Marked EXPERIMENT in the source, so treat the shape as not yet frozen.
// @Tags			catalog
// @Produce		json
// @Param			room_id_or_alias	path		string				true	"Room ID or alias"
// @Param			via					query		string				false	"Server to try for the MSC3266 fallback if we do not have the room indexed"
// @Success		200					{object}	model.MatrixRoom	"Room data"
// @Failure		404					{object}	model.MatrixError	"Room not found anywhere we looked"
// @Failure		500					{object}	model.MatrixError	"Internal error reading the room"
// @Router			/room/{room_id_or_alias} [get]
func catalogRoom(dataSvc dataService, matrixSvc matrixService, plausible plausibleService) echo.HandlerFunc {
	return func(c echo.Context) error {
		roomIDorAlias := utils.Unescape(c.Param("room_id_or_alias"))
		if !utils.IsValidID(roomIDorAlias) && utils.IsValidAlias("#"+roomIDorAlias) {
			roomIDorAlias = "#" + roomIDorAlias
		}

		// 1. Try to get the room directly from the database
		room, err := dataSvc.GetRoom(c.Request().Context(), roomIDorAlias)
		if err != nil {
			return c.JSONBlob(http.StatusInternalServerError, utils.MustJSON(model.MatrixError{
				Code:    "M_INTERNAL_SERVER_ERROR",
				Message: err.Error(),
			}))
		}

		// 2. If the room is not found, try to get it using MSC3266
		if room == nil {
			_, entry := matrixSvc.GetClientRoomSummary(c.Request().Context(), roomIDorAlias, c.QueryParam("via"), true)
			if entry != nil {
				room = entry.Convert("")
				c.Response().Header().Set("X-MRS-MSC3266", "true")
			}
		}

		// 3. If the room is still not found, return 404
		if room == nil {
			evt := model.NewAnalyticsEvent(c.Request().Context(), "OpenNotFound", map[string]string{"room": roomIDorAlias}, c.Request())
			go func(ctx context.Context, evt *model.AnalyticsEvent) {
				ctx = context.WithoutCancel(ctx)
				plausible.Track(ctx, evt)
			}(c.Request().Context(), evt)

			return c.JSONBlob(http.StatusNotFound, utils.MustJSON(model.MatrixError{
				Code:    "M_NOT_FOUND",
				Message: "room not found",
			}))
		}

		evt := model.NewAnalyticsEvent(c.Request().Context(), "Open", map[string]string{"room": room.Alias}, c.Request())
		go func(ctx context.Context, evt *model.AnalyticsEvent) {
			ctx = context.WithoutCancel(ctx)
			plausible.Track(ctx, evt)
		}(c.Request().Context(), evt)

		return c.JSON(http.StatusOK, room)
	}
}

// @Summary		All rooms
// @Description	Every indexed room as a room-ID to alias map. Big, authenticated, and exactly as heavy as it sounds.
// @Tags			catalog
// @Produce		json
// @Security		CatalogAuth
// @Success		200	{object}	map[string]string	"Room ID to alias"
// @Router			/catalog/rooms [get]
func rooms(data dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		rooms := map[string]string{}
		data.EachRoom(c.Request().Context(), func(roomID string, room *model.MatrixRoom) bool {
			if room == nil {
				return false
			}
			rooms[roomID] = room.Alias
			return false
		})
		return c.JSON(http.StatusOK, rooms)
	}
}

// @Summary		Online servers
// @Description	The servers the crawler currently considers online.
// @Tags			catalog
// @Produce		json
// @Security		CatalogAuth
// @Success		200	{array}	string	"Online server names"
// @Router			/catalog/servers [get]
func servers(crawler crawlerService) echo.HandlerFunc {
	return func(c echo.Context) error {
		servers := crawler.OnlineServers(c.Request().Context())
		return c.JSON(http.StatusOK, servers)
	}
}

// @Summary		Online servers with details
// @Description	Online servers with their full crawl records, keyed by server name.
// @Tags			catalog
// @Produce		json
// @Security		CatalogAuth
// @Success		200	{object}	map[string]model.MatrixServer	"Server name to crawl record"
// @Router			/catalog/servers/objects [get]
func serversObjects(crawler crawlerService) echo.HandlerFunc {
	return func(c echo.Context) error {
		servers := crawler.OnlineServersObjects(c.Request().Context())
		return c.JSON(http.StatusOK, servers)
	}
}
