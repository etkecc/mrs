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

func catalogServers(dataSvc dataService) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, dataSvc.GetServersRoomsCount(c.Request().Context()))
	}
}
