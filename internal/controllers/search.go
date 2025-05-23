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
	Search(ctx context.Context, req *http.Request, query, sortBy string, limit, offset int) ([]*model.Entry, int, error)
}

func search(svc searchService, cfg configService, path bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer metrics.IncSearchQueries("rest", cfg.Get().Matrix.ServerName)

		paramfunc := c.QueryParam
		if path {
			paramfunc = c.Param
		}

		query := utils.Unescape(paramfunc("q"))

		limit := kit.StringToInt(paramfunc("l"))
		offset := kit.StringToInt(paramfunc("o"))
		sortBy := paramfunc("s")
		entries, _, err := svc.Search(c.Request().Context(), c.Request(), query, sortBy, limit, offset)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return c.NoContent(http.StatusNoContent)
		}
		return c.JSON(http.StatusOK, entries)
	}
}
