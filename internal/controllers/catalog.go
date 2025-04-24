package controllers

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

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
			respb, err := utils.JSON(model.MatrixError{
				Code:    "M_INVALID_PARAM",
				Message: "invalid alias",
			})
			if err != nil {
				zerolog.Ctx(c.Request().Context()).Error().Err(err).Msg("cannot marshal canonical json")
			}
			return c.JSONBlob(http.StatusBadRequest, respb)
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
		if utils.IsValidAlias("#" + roomIDorAlias) {
			roomIDorAlias = "#" + roomIDorAlias
		}

		room, err := dataSvc.GetRoom(c.Request().Context(), roomIDorAlias)
		if err != nil {
			respb, jerr := utils.JSON(model.MatrixError{
				Code:    "M_INTERNAL_SERVER_ERROR",
				Message: err.Error(),
			})
			if jerr != nil {
				zerolog.Ctx(c.Request().Context()).Error().Err(jerr).Msg("cannot marshal canonical json")
			}
			return c.JSONBlob(http.StatusBadRequest, respb)
		}

		if room == nil {
			respb, err := utils.JSON(model.MatrixError{
				Code:    "M_NOT_FOUND",
				Message: "room not found",
			})
			if err != nil {
				zerolog.Ctx(c.Request().Context()).Error().Err(err).Msg("cannot marshal canonical json")
			}
			return c.JSONBlob(http.StatusNotFound, respb)
		}

		return c.JSON(http.StatusOK, room)
	}
}
