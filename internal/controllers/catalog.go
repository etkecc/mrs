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
}

// catalogRoom returns the room data for the given room ID or alias.
// EXPERIMENT! This endpoint returns the room data for the given room ID or alias.
// similar to the room preview endpoint from Matrix CS API, but using all MRS' room properties (like language, etc)
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
				room = entry.Convert()
				c.Response().Header().Set("X-MRS-MSC3266", "true")
			}
		}

		// 3. If the room is still not found, return 404
		if room == nil {
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

// rooms returns a list of all rooms in the database.
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

// servers returns a list of online servers that the crawler is aware of.
func servers(crawler crawlerService) echo.HandlerFunc {
	return func(c echo.Context) error {
		servers := crawler.OnlineServers(c.Request().Context())
		return c.JSON(http.StatusOK, servers)
	}
}
