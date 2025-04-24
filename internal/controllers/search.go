package controllers

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/etkecc/go-kit"
	"github.com/etkecc/mrs/internal/metrics"
	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
)

type searchService interface {
	Search(ctx context.Context, originServer, query, sortBy string, limit, offset int) ([]*model.Entry, int, error)
}

func search(svc searchService, plausible plausibleService, cfg configService, path bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		origin := getOrigin(cfg, c.Request())
		defer metrics.IncSearchQueries("rest", origin)

		paramfunc := c.QueryParam
		if path {
			paramfunc = c.Param
		}

		query := utils.Unescape(paramfunc("q"))
		go func(req *http.Request, ip, query string) {
			ctx := context.WithoutCancel(req.Context())
			plausible.TrackSearch(ctx, req, ip, query)
		}(c.Request(), c.RealIP(), query)

		limit := kit.StringToInt(paramfunc("l"))
		offset := kit.StringToInt(paramfunc("o"))
		sortBy := paramfunc("s")
		entries, _, err := svc.Search(c.Request().Context(), origin, query, sortBy, limit, offset)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return c.NoContent(http.StatusNoContent)
		}
		return c.JSON(http.StatusOK, entries)
	}
}
