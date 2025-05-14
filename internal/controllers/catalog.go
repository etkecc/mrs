package controllers

import (
	"context"
	"net/http"

	"github.com/etkecc/go-apm"
	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

// openRoom redirects to the matrix.to link for the given room alias.
// EXPERIMENT! This is a mere wrapper with plausible tracking.
// If it will work as expected (i.e., storing the room open event in plausible),
// later we can extend it to have something like "popular rooms" endpoint (e.g., /catalog/popular)
func openRoom(plausible plausibleService) echo.HandlerFunc {
	return func(c echo.Context) error {
		alias := "#" + utils.Unescape(c.Param("room_alias"))
		if !utils.IsValidAlias(alias) {
			return c.JSONBlob(http.StatusBadRequest, utils.MustJSON(model.MatrixError{
				Code:    "M_INVALID_PARAM",
				Message: "invalid alias",
			}))
		}

		go func(req *http.Request, ip, alias string) {
			ctx := context.WithoutCancel(req.Context())
			plausible.TrackOpen(ctx, req, ip, alias)
		}(c.Request(), c.RealIP(), alias)

		return c.Redirect(http.StatusFound, "https://matrix.to/#/"+alias)
	}
}

// catalogServers returns the count of rooms for each server
// EXPERIMENT! This is a mere wrapper for the data service.
func catalogServers(dataSvc dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, dataSvc.GetServersRoomsCount(c.Request().Context()))
	}
}

// catalogRoom returns the room data for the given room ID or alias.
// EXPERIMENT! This endpoint returns the room data for the given room ID or alias.
// similar to the room preview endpoint from Matrix CS API, but using all MRS' room properties (like language, etc)
func catalogRoom(dataSvc dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		roomIDorAlias := utils.Unescape(c.Param("room_id_or_alias"))
		if !utils.IsValidID(roomIDorAlias) && utils.IsValidAlias("#"+roomIDorAlias) {
			roomIDorAlias = "#" + roomIDorAlias
		}

		apm.Log(c.Request().Context()).Info().Str("room_id_or_alias", roomIDorAlias).Msg("catalogRoom")
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

		return c.JSON(http.StatusOK, room)
	}
}
