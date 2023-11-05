package controllers

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"

	"gitlab.com/etke.cc/mrs/api/metrics"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

type searchService interface {
	Search(query, sortBy string, limit, offset int) ([]*model.Entry, int, error)
}

func search(svc searchService, cfg configService, path bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer metrics.IncSearchQueries("rest", cfg.Get().Matrix.ServerName)

		paramfunc := c.QueryParam
		if path {
			paramfunc = c.Param
		}

		query, err := url.QueryUnescape(paramfunc("q"))
		if err != nil {
			return err
		}
		limit := utils.StringToInt(paramfunc("l"))
		offset := utils.StringToInt(paramfunc("o"))
		sortBy := paramfunc("s")
		entries, _, err := svc.Search(query, sortBy, limit, offset)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return c.NoContent(http.StatusNoContent)
		}
		return c.JSON(http.StatusOK, entries)
	}
}
