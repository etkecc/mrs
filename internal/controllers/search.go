package controllers

import (
	"context"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"

	"github.com/etkecc/mrs/internal/metrics"
	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

type searchService interface {
	Search(ctx context.Context, query, sortBy string, limit, offset int) ([]*model.Entry, int, error)
}

func search(svc searchService, plausible plausibleService, cfg configService, path bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer metrics.IncSearchQueries("rest", getOrigin(cfg, c.Request()))

		paramfunc := c.QueryParam
		if path {
			paramfunc = c.Param
		}

		query, err := url.QueryUnescape(paramfunc("q"))
		if err != nil {
			return err
		}
		go plausible.TrackSearch(c.Request().Context(), c.Request(), c.RealIP(), query)

		limit := utils.StringToInt(paramfunc("l"))
		offset := utils.StringToInt(paramfunc("o"))
		sortBy := paramfunc("s")
		entries, _, err := svc.Search(c.Request().Context(), query, sortBy, limit, offset)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return c.NoContent(http.StatusNoContent)
		}
		return c.JSON(http.StatusOK, entries)
	}
}
