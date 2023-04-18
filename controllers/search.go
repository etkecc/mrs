package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"gitlab.com/etke.cc/mrs/api/model"
)

type searchService interface {
	Search(query string, limit, offset int) ([]*model.Entry, error)
}

const (
	DefaultSearchLimit  = 10
	DefaultSearchOffset = 0
)

func search(svc searchService, path bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		paramfunc := c.QueryParam
		if path {
			paramfunc = c.Param
		}

		query := paramfunc("q")
		limit := string2int(paramfunc("l"), DefaultSearchLimit)
		offset := string2int(paramfunc("o"), DefaultSearchOffset)
		entries, err := svc.Search(query, limit, offset)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return c.NoContent(http.StatusNoContent)
		}
		return c.JSON(http.StatusOK, entries)
	}
}
