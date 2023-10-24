package controllers

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"

	"gitlab.com/etke.cc/mrs/api/model"
	"gitlab.com/etke.cc/mrs/api/utils"
)

type searchService interface {
	Search(query string, limit, offset int, sortBy []string) ([]*model.Entry, error)
}

const (
	DefaultSearchLimit  = 10
	DefaultSearchOffset = 0
	DefaultSearchSortBy = "-members,-_score"
)

func search(svc searchService, path bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		paramfunc := c.QueryParam
		if path {
			paramfunc = c.Param
		}

		query, err := url.QueryUnescape(paramfunc("q"))
		if err != nil {
			return err
		}
		limit := utils.StringToInt(paramfunc("l"), DefaultSearchLimit)
		offset := utils.StringToInt(paramfunc("o"), DefaultSearchOffset)
		sortBy := utils.StringToSlice(paramfunc("s"), DefaultSearchSortBy)
		entries, err := svc.Search(query, limit, offset, sortBy)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return c.NoContent(http.StatusNoContent)
		}
		return c.JSON(http.StatusOK, entries)
	}
}
