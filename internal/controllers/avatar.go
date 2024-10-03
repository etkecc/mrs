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

		avatar, contentType := svc.GetMediaThumbnail(c.Request().Context(), name, id, c.QueryParams())
		if contentType == "" {
			return c.NoContent(http.StatusNoContent)
		}

		return c.Stream(http.StatusOK, contentType, avatar)
	}
}
