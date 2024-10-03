package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func avatar(svc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("name")
		id := c.Param("id")
		if name == "" || id == "" {
			return c.NoContent(http.StatusNoContent)
		}

		// attempt to get unauthenticated media thumbnail first (CS API, faster)
		avatar, contentType := svc.GetClientMediaThumbnail(c.Request().Context(), name, id, c.QueryParams())
		if contentType != "" {
			return c.Stream(http.StatusOK, contentType, avatar)
		}

		// fallback to authenticated media thumbnail (S2S API, slower)
		avatar, contentType = svc.GetMediaThumbnail(c.Request().Context(), name, id, c.QueryParams())
		if contentType != "" {
			return c.Stream(http.StatusOK, contentType, avatar)
		}

		return c.NoContent(http.StatusNoContent)
	}
}
