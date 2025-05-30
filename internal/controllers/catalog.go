package controllers

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

// catalogRoom returns the room data for the given room ID or alias.
// EXPERIMENT! This endpoint returns the room data for the given room ID or alias.
// similar to the room preview endpoint from Matrix CS API, but using all MRS' room properties (like language, etc)
func catalogRoom(dataSvc dataService, plausible plausibleService) echo.HandlerFunc {
	return func(c echo.Context) error {
		roomIDorAlias := utils.Unescape(c.Param("room_id_or_alias"))
		if !utils.IsValidID(roomIDorAlias) && utils.IsValidAlias("#"+roomIDorAlias) {
			roomIDorAlias = "#" + roomIDorAlias
		}

		room, err := dataSvc.GetRoom(c.Request().Context(), roomIDorAlias)
		if err != nil {
			return c.JSONBlob(http.StatusInternalServerError, utils.MustJSON(model.MatrixError{
				Code:    "M_INTERNAL_SERVER_ERROR",
				Message: err.Error(),
			}))
		}

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
