package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// @Summary		Media thumbnail
// @Description	Serves a media thumbnail. Tries the unauthenticated client-server media API first (faster), then falls back to the authenticated federation media API. Streams the image on a hit, or 204 when there is nothing to serve. Standard Matrix thumbnail query params (width, height, method) are passed straight through.
// @Tags			media
// @Produce		image/jpeg
// @Produce		image/png
// @Param			name	path	string	true	"Media origin server name"
// @Param			id		path	string	true	"Media ID"
// @Param			width	query	int		false	"Thumbnail width"
// @Param			height	query	int		false	"Thumbnail height"
// @Param			method	query	string	false	"Resize method: crop or scale"
// @Success		200		{file}	binary	"Thumbnail image"
// @Success		204		"No thumbnail available"
// @Router			/avatar/{name}/{id} [get]
// @Router			/_matrix/media/r0/thumbnail/{name}/{id} [get]
// @Router			/_matrix/media/v3/thumbnail/{name}/{id} [get]
func avatar(svc matrixService) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("name")
		id := c.Param("id")
		if name == "" || id == "" {
			return c.NoContent(http.StatusNoContent)
		}

		// attempt to get unauthenticated media thumbnail first (CS API, faster)
		avatar, contentType := svc.GetClientMediaThumbnail(c.Request().Context(), name, id, c.QueryParams())
		if avatar != nil && contentType != "" {
			return c.Stream(http.StatusOK, contentType, avatar)
		}

		// fallback to authenticated media thumbnail (S2S API, slower)
		avatar, contentType = svc.GetMediaThumbnail(c.Request().Context(), name, id, c.QueryParams())
		if avatar != nil && contentType != "" {
			return c.Stream(http.StatusOK, contentType, avatar)
		}

		return c.NoContent(http.StatusNoContent)
	}
}
