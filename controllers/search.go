package controllers

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/etke.cc/mrs/api/metrics"
	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

type searchService interface {
	Search(query, sortBy string, limit, offset int) ([]*model.Entry, int, error)
}

func search(svc searchService, serverName string, path bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer metrics.SearchQueries.
			With(prometheus.Labels{
				"api":    "rest",
				"server": serverName,
			}).Inc()

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
