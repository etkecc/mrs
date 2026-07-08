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
	Search(ctx context.Context, req *http.Request, query, sortBy string, roomTypes []string, limit, offset int) ([]*model.Entry, int, error)
}

// @Summary		Search rooms
// @Description	Full-text room search. Pass the query and options as query params here: ?q= (query), ?l= (limit), ?o= (offset), ?s= (sort), ?rt= (room type). The same handler also answers a positional path form for convenience, /search/{q}, /search/{q}/{l}, and so on up to /search/{q}/{l}/{o}/{s}/{rt}, filling those five slots left to right. An empty result set is a 204, not an empty 200.
// @Tags			search
// @Produce		json
// @Param			q	query	string		false	"Search query"
// @Param			l	query	int			false	"Limit"
// @Param			o	query	int			false	"Offset"
// @Param			s	query	string		false	"Sort field"
// @Param			rt	query	string		false	"Room type filter"
// @Success		200	{array}	model.Entry	"Matching rooms"
// @Success		204	"No matches"
// @Router			/search [get]
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
		roomType := utils.Unescape(paramfunc("rt"))
		var roomTypes []string
		if roomType != "" {
			roomTypes = []string{roomType}
		}
		entries, _, err := svc.Search(c.Request().Context(), c.Request(), query, sortBy, roomTypes, limit, offset)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return c.NoContent(http.StatusNoContent)
		}
		return c.JSON(http.StatusOK, entries)
	}
}
